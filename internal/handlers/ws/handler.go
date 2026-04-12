package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/getchatapi/chatapi/internal/auth"
	"github.com/getchatapi/chatapi/internal/config"
	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/ratelimit"
	"github.com/getchatapi/chatapi/internal/services/bot"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/services/delivery"
	"github.com/getchatapi/chatapi/internal/services/message"
	"github.com/getchatapi/chatapi/internal/services/realtime"
)

// Handler handles WebSocket connections
type Handler struct {
	chatroomSvc *chatroom.Service
	messageSvc  *message.Service
	realtimeSvc *realtime.Service
	deliverySvc *delivery.Service
	botSvc      *bot.Service
	msgLimiter  *ratelimit.Limiter // nil = disabled
	jwtSecret   string
	upgrader    websocket.Upgrader
}

// NewHandler creates a new WebSocket handler
func NewHandler(
	chatroomSvc *chatroom.Service,
	messageSvc *message.Service,
	realtimeSvc *realtime.Service,
	deliverySvc *delivery.Service,
	botSvc *bot.Service,
	cfg *config.Config,
	msgLimiter *ratelimit.Limiter,
) *Handler {
	allowedOrigins := cfg.AllowedOrigins

	if len(allowedOrigins) == 0 {
		slog.Warn("ALLOWED_ORIGINS is not set — WebSocket connections will be rejected for browser clients sending an Origin header. Set to \"*\" to allow all origins (dev only).")
	}

	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}

	checkOrigin := func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if _, ok := originSet["*"]; ok {
			return true
		}
		if _, ok := originSet[origin]; ok {
			return true
		}
		slog.Warn("WebSocket connection rejected: origin not allowed",
			"origin", origin,
			"remote_addr", r.RemoteAddr)
		return false
	}

	return &Handler{
		chatroomSvc: chatroomSvc,
		messageSvc:  messageSvc,
		realtimeSvc: realtimeSvc,
		deliverySvc: deliverySvc,
		botSvc:      botSvc,
		msgLimiter:  msgLimiter,
		jwtSecret:   cfg.JWTSecret,
		upgrader:    websocket.Upgrader{CheckOrigin: checkOrigin},
	}
}

// HandleConnection handles WebSocket connections.
//
// Auth: JWT is accepted as:
//   - ?token=<jwt>  — for browser clients (cannot set custom headers on WS upgrade)
//   - Authorization: Bearer <jwt>  — for server-to-server clients
func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}

	if err := h.realtimeSvc.RegisterConnection(userID, conn); err != nil {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "connection limit reached"))
		conn.Close()
		return
	}

	h.realtimeSvc.BroadcastPresenceUpdate(userID, "online")

	go h.handleReconnectSync(userID, conn)
	go h.handleConnection(userID, conn)
}

// authenticate extracts and validates the JWT from the request.
// Returns the user ID and true on success; writes the error response and returns false on failure.
func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	var tokenStr string

	if t := r.URL.Query().Get("token"); t != "" {
		tokenStr = t
	} else if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
		tokenStr = strings.TrimPrefix(hdr, "Bearer ")
	}

	if tokenStr == "" {
		http.Error(w, "Missing authentication", http.StatusUnauthorized)
		return "", false
	}

	userID, err := auth.ValidateJWT(h.jwtSecret, tokenStr)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return "", false
	}

	return userID, true
}

// handleReconnectSync sends missed messages to a reconnecting client.
// Currently a no-op — clients request missed messages via after_seq on reconnect.
func (h *Handler) handleReconnectSync(userID string, conn *websocket.Conn) {}

// handleConnection processes messages from a WebSocket connection
func (h *Handler) handleConnection(userID string, conn *websocket.Conn) {
	defer func() {
		h.realtimeSvc.UnregisterConnection(userID, conn)
		conn.Close()
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket error", "user_id", userID, "error", err)
			}
			break
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			slog.Warn("Invalid WebSocket message", "user_id", userID, "error", err)
			continue
		}

		if wsMsg.Type == "send_message" && h.msgLimiter != nil && !h.msgLimiter.Allow(userID) {
			conn.WriteJSON(map[string]interface{}{
				"type": "error",
				"data": map[string]interface{}{
					"code":    "rate_limited",
					"message": "too many requests",
				},
			})
			continue
		}

		if err := h.handleMessage(userID, &wsMsg); err != nil {
			slog.Error("Failed to handle WebSocket message",
				"user_id", userID,
				"type", wsMsg.Type,
				"error", err)
		}
	}
}

// handleMessage processes different types of WebSocket messages
func (h *Handler) handleMessage(userID string, msg *models.WSMessage) error {
	switch msg.Type {
	case "send_message":
		return h.handleSendMessage(userID, msg.Data)
	case "ack":
		return h.handleAck(userID, msg.Data)
	case "typing.start":
		return h.handleTyping(userID, msg.Data, "start")
	case "typing.stop":
		return h.handleTyping(userID, msg.Data, "stop")
	case "ping":
		return nil
	default:
		slog.Warn("Unknown message type", "type", msg.Type, "user_id", userID)
		return nil
	}
}

// handleSendMessage handles message sending via WebSocket
func (h *Handler) handleSendMessage(userID string, data interface{}) error {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	roomID, ok := msgData["room_id"].(string)
	if !ok {
		return nil
	}

	content, ok := msgData["content"].(string)
	if !ok {
		return nil
	}

	req := &models.CreateMessageRequest{Content: content}
	if meta, ok := msgData["meta"].(string); ok {
		req.Meta = meta
	}

	message, err := h.messageSvc.SendMessage(roomID, userID, req)
	if err != nil {
		return err
	}

	broadcast := map[string]interface{}{
		"type":       "message",
		"room_id":    roomID,
		"seq":        message.Seq,
		"message_id": message.MessageID,
		"sender_id":  message.SenderID,
		"content":    message.Content,
		"created_at": message.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if message.Meta != "" {
		broadcast["meta"] = message.Meta
	}
	h.realtimeSvc.BroadcastToRoom(roomID, broadcast)

	go h.deliverySvc.HandleNewMessage(roomID, message)

	// Trigger managed bots. Bots do not trigger other bots.
	if !h.botSvc.IsBot(userID) {
		go h.botSvc.TriggerBots(context.Background(), roomID, message, h.realtimeSvc)
	}

	return nil
}

// handleAck handles acknowledgment of message delivery
func (h *Handler) handleAck(userID string, data interface{}) error {
	ackData, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	roomID, ok := ackData["room_id"].(string)
	if !ok {
		return nil
	}

	seqFloat, ok := ackData["seq"].(float64)
	if !ok {
		return nil
	}

	if err := h.messageSvc.UpdateLastAck(userID, roomID, int(seqFloat)); err != nil {
		return err
	}

	h.realtimeSvc.BroadcastToRoom(roomID, map[string]interface{}{
		"type":    "ack.received",
		"room_id": roomID,
		"seq":     int(seqFloat),
		"user_id": userID,
	})

	return nil
}

// handleTyping handles typing indicators
func (h *Handler) handleTyping(userID string, data interface{}, action string) error {
	typingData, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	roomID, ok := typingData["room_id"].(string)
	if !ok {
		return nil
	}

	h.realtimeSvc.BroadcastToRoom(roomID, map[string]interface{}{
		"type":    "typing",
		"room_id": roomID,
		"user_id": userID,
		"action":  action,
	})

	return nil
}
