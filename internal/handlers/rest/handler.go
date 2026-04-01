package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hastenr/chatapi/internal/auth"
	"github.com/hastenr/chatapi/internal/config"
	"github.com/hastenr/chatapi/internal/models"
	"github.com/hastenr/chatapi/internal/services/chatroom"
	"github.com/hastenr/chatapi/internal/services/delivery"
	"github.com/hastenr/chatapi/internal/services/message"
	"github.com/hastenr/chatapi/internal/services/notification"
	"github.com/hastenr/chatapi/internal/services/realtime"
)

// defaultTenantID is used throughout as ChatAPI is single-tenant per deployment.
const defaultTenantID = "default"

// Handler handles REST API requests
type Handler struct {
	chatroomSvc *chatroom.Service
	messageSvc  *message.Service
	realtimeSvc *realtime.Service
	deliverySvc *delivery.Service
	notifSvc    *notification.Service
	jwtSecret   string
	startTime   time.Time
}

// NewHandler creates a new REST handler
func NewHandler(
	chatroomSvc *chatroom.Service,
	messageSvc *message.Service,
	realtimeSvc *realtime.Service,
	deliverySvc *delivery.Service,
	notifSvc *notification.Service,
	cfg *config.Config,
) *Handler {
	return &Handler{
		chatroomSvc: chatroomSvc,
		messageSvc:  messageSvc,
		realtimeSvc: realtimeSvc,
		deliverySvc: deliverySvc,
		notifSvc:    notifSvc,
		jwtSecret:   cfg.JWTSecret,
		startTime:   time.Now(),
	}
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

// AuthMiddleware validates the Bearer JWT and stores the user ID in context.
func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, "unauthorized", "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		userID, err := auth.ValidateJWT(h.jwtSecret, tokenStr)
		if err != nil {
			writeError(w, "unauthorized", "Invalid token", http.StatusUnauthorized)
			return
		}

		r = r.WithContext(auth.WithUserID(r.Context(), userID))
		next(w, r)
	}
}

// getUserID retrieves the authenticated user ID from the request context.
func (h *Handler) getUserID(r *http.Request) string {
	uid, _ := auth.UserIDFromContext(r.Context())
	return uid
}

// requireUserID returns the user ID or writes a 401 and returns "".
// Since AuthMiddleware guarantees the user ID is set, this is a safety net.
func (h *Handler) requireUserID(w http.ResponseWriter, r *http.Request) string {
	uid := h.getUserID(r)
	if uid == "" {
		writeError(w, "unauthorized", "Unauthorized", http.StatusUnauthorized)
	}
	return uid
}

// HandleHealth health check endpoint
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "chatapi",
		"uptime":  uptime.String(),
	})
}

// HandleMetrics exposes operational counters for monitoring.
// All values are process-lifetime totals (reset on restart) except active_connections.
func (h *Handler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_connections": h.realtimeSvc.ActiveConnections(),
		"broadcast_drops":    h.realtimeSvc.DroppedBroadcasts(),
		"messages_sent":      h.messageSvc.MessagesSent(),
		"delivery_attempts":  h.deliverySvc.DeliveryAttempts(),
		"delivery_failures":  h.deliverySvc.DeliveryFailures(),
		"uptime_seconds":     int64(time.Since(h.startTime).Seconds()),
	})
}

// HandleCreateRoom create room endpoint
func (h *Handler) HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	var req models.CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}

	room, err := h.chatroomSvc.CreateRoom(defaultTenantID, &req)
	if err != nil {
		slog.Error("Failed to create room", "error", err, "user_id", userID)
		writeError(w, "internal_error", "Failed to create room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// HandleGetRoom get room endpoint
func (h *Handler) HandleGetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")

	room, err := h.chatroomSvc.GetRoom(defaultTenantID, roomID)
	if err != nil {
		writeError(w, "not_found", "Room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// HandleGetRoomMembers get room members endpoint
func (h *Handler) HandleGetRoomMembers(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")

	members, err := h.chatroomSvc.GetRoomMembers(defaultTenantID, roomID)
	if err != nil {
		writeError(w, "internal_error", "Failed to get room members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"members": members,
	})
}

// HandleSendMessage send message endpoint
func (h *Handler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	roomID := r.PathValue("room_id")

	var req models.CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		writeError(w, "invalid_request", "content is required", http.StatusBadRequest)
		return
	}

	message, err := h.messageSvc.SendMessage(defaultTenantID, roomID, userID, &req)
	if err != nil {
		slog.Error("Failed to send message", "error", err, "user_id", userID, "room_id", roomID)
		writeError(w, "internal_error", "Failed to send message", http.StatusInternalServerError)
		return
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
	h.realtimeSvc.BroadcastToRoom(defaultTenantID, roomID, broadcast)

	go h.deliverySvc.HandleNewMessage(defaultTenantID, roomID, message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// HandleGetMessages get messages endpoint
func (h *Handler) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")

	afterSeq := 0
	if after := r.URL.Query().Get("after_seq"); after != "" {
		if seq, err := strconv.Atoi(after); err == nil {
			afterSeq = seq
		}
	}

	limit := 50
	if lim := r.URL.Query().Get("limit"); lim != "" {
		if l, err := strconv.Atoi(lim); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	messages, err := h.messageSvc.GetMessages(defaultTenantID, roomID, afterSeq, limit)
	if err != nil {
		writeError(w, "internal_error", "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}

// HandleAck ACK endpoint
func (h *Handler) HandleAck(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	var req models.AckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.messageSvc.UpdateLastAck(defaultTenantID, userID, req.RoomID, req.Seq); err != nil {
		writeError(w, "internal_error", "Failed to process acknowledgment", http.StatusInternalServerError)
		return
	}

	h.realtimeSvc.BroadcastToRoom(defaultTenantID, req.RoomID, map[string]interface{}{
		"type":    "ack.received",
		"room_id": req.RoomID,
		"seq":     req.Seq,
		"user_id": userID,
	})

	w.WriteHeader(http.StatusOK)
}

// HandleNotify notify endpoint
func (h *Handler) HandleNotify(w http.ResponseWriter, r *http.Request) {
	var req models.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}

	notif, err := h.notifSvc.CreateNotification(defaultTenantID, &req)
	if err != nil {
		slog.Error("Failed to create notification", "error", err)
		writeError(w, "internal_error", "Failed to create notification", http.StatusInternalServerError)
		return
	}

	go h.deliverySvc.DeliverNow(notif)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(notif)
}

// HandleGetDeadLetters admin endpoint to get failed deliveries
func (h *Handler) HandleGetDeadLetters(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if lim := r.URL.Query().Get("limit"); lim != "" {
		if l, err := strconv.Atoi(lim); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	failedNotifications, err := h.notifSvc.GetFailedNotifications(defaultTenantID, limit)
	if err != nil {
		writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
		return
	}

	failedMessages, err := h.messageSvc.GetFailedUndeliveredMessages(defaultTenantID, limit)
	if err != nil {
		writeError(w, "internal_error", err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"failed_notifications": failedNotifications,
		"failed_messages":      failedMessages,
	})
}

// HandleGetUserRooms returns all rooms the authenticated user belongs to
func (h *Handler) HandleGetUserRooms(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	rooms, err := h.chatroomSvc.GetUserRooms(defaultTenantID, userID)
	if err != nil {
		slog.Error("Failed to get user rooms", "error", err, "user_id", userID)
		writeError(w, "internal_error", "Failed to get rooms", http.StatusInternalServerError)
		return
	}

	if rooms == nil {
		rooms = []*models.Room{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"rooms": rooms})
}

// HandleSubscribe subscribes the authenticated user to a notification topic
func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	var req struct {
		Topic string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Topic == "" {
		writeError(w, "invalid_request", "topic is required", http.StatusBadRequest)
		return
	}

	sub, err := h.notifSvc.Subscribe(defaultTenantID, userID, req.Topic)
	if err != nil {
		slog.Error("Failed to subscribe", "error", err, "user_id", userID)
		writeError(w, "internal_error", "Failed to subscribe", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

// HandleUnsubscribe removes a notification subscription by ID
func (h *Handler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "invalid_request", "Invalid subscription ID", http.StatusBadRequest)
		return
	}

	if err := h.notifSvc.Unsubscribe(defaultTenantID, userID, id); err != nil {
		writeError(w, "not_found", "Subscription not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleListSubscriptions lists the authenticated user's notification subscriptions
func (h *Handler) HandleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	subs, err := h.notifSvc.GetUserSubscriptions(defaultTenantID, userID)
	if err != nil {
		writeError(w, "internal_error", "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	if subs == nil {
		subs = []*models.NotificationSubscription{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"subscriptions": subs})
}

// HandleDeleteMessage deletes a message. Only the original sender may delete their own message.
func (h *Handler) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	roomID := r.PathValue("room_id")
	messageID := r.PathValue("message_id")

	seq, err := h.messageSvc.DeleteMessage(defaultTenantID, roomID, messageID, userID)
	if err != nil {
		switch err.Error() {
		case "message not found":
			writeError(w, "not_found", "Message not found", http.StatusNotFound)
		case "forbidden":
			writeError(w, "forbidden", "You can only delete your own messages", http.StatusForbidden)
		default:
			slog.Error("Failed to delete message", "error", err, "message_id", messageID)
			writeError(w, "internal_error", "Failed to delete message", http.StatusInternalServerError)
		}
		return
	}

	h.realtimeSvc.BroadcastToRoom(defaultTenantID, roomID, map[string]interface{}{
		"type":       "message.deleted",
		"room_id":    roomID,
		"message_id": messageID,
		"seq":        seq,
	})

	w.WriteHeader(http.StatusNoContent)
}

// HandleUpdateRoom updates a room's name and/or metadata.
func (h *Handler) HandleUpdateRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")

	var req models.UpdateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}

	room, err := h.chatroomSvc.UpdateRoom(defaultTenantID, roomID, &req)
	if err != nil {
		if err.Error() == "room not found" {
			writeError(w, "not_found", "Room not found", http.StatusNotFound)
			return
		}
		slog.Error("Failed to update room", "error", err, "room_id", roomID)
		writeError(w, "internal_error", "Failed to update room", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// HandleEditMessage updates the content of a message. Only the original sender may edit.
func (h *Handler) HandleEditMessage(w http.ResponseWriter, r *http.Request) {
	userID := h.requireUserID(w, r)
	if userID == "" {
		return
	}

	roomID := r.PathValue("room_id")
	messageID := r.PathValue("message_id")

	var req models.UpdateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid_request", "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		writeError(w, "invalid_request", "content is required", http.StatusBadRequest)
		return
	}

	msg, err := h.messageSvc.UpdateMessage(defaultTenantID, roomID, messageID, userID, req.Content)
	if err != nil {
		switch err.Error() {
		case "message not found":
			writeError(w, "not_found", "Message not found", http.StatusNotFound)
		case "forbidden":
			writeError(w, "forbidden", "You can only edit your own messages", http.StatusForbidden)
		default:
			slog.Error("Failed to edit message", "error", err, "message_id", messageID)
			writeError(w, "internal_error", "Failed to edit message", http.StatusInternalServerError)
		}
		return
	}

	h.realtimeSvc.BroadcastToRoom(defaultTenantID, roomID, map[string]interface{}{
		"type":       "message.edited",
		"room_id":    roomID,
		"message_id": messageID,
		"content":    msg.Content,
		"seq":        msg.Seq,
		"sender_id":  msg.SenderID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}
