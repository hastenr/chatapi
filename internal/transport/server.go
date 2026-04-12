package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/getchatapi/chatapi/internal/auth"
	"github.com/getchatapi/chatapi/internal/config"
	"github.com/getchatapi/chatapi/internal/db"
	"github.com/getchatapi/chatapi/internal/handlers/rest"
	"github.com/getchatapi/chatapi/internal/handlers/ws"
	"github.com/getchatapi/chatapi/internal/ratelimit"
	"github.com/getchatapi/chatapi/internal/repository/sqlite"
	"github.com/getchatapi/chatapi/internal/services/bot"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/services/delivery"
	"github.com/getchatapi/chatapi/internal/services/message"
	"github.com/getchatapi/chatapi/internal/services/realtime"
	"github.com/getchatapi/chatapi/internal/services/webhook"
)

// Server represents the HTTP server
type Server struct {
	httpServer  *http.Server
	config      *config.Config
	realtimeSvc *realtime.Service
	msgLimiter  *ratelimit.Limiter
}

// NewServer creates and wires up the HTTP server
func NewServer(cfg *config.Config, database *db.DB, realtimeSvc *realtime.Service) *Server {
	// 1. Repositories
	roomRepo := sqlite.NewRoomRepository(database.DB)
	msgRepo := sqlite.NewMessageRepository(database.DB)
	delivRepo := sqlite.NewDeliveryRepository(database.DB)
	botRepo := sqlite.NewBotRepository(database.DB)

	// 2. Services (order matters — later services depend on earlier ones)
	chatroomSvc := chatroom.NewService(roomRepo)
	messageSvc := message.NewService(msgRepo)
	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(delivRepo, realtimeSvc, chatroomSvc, cfg.WebhookURL, cfg.WebhookSecret, webhookSvc)
	botSvc := bot.NewService(botRepo, msgRepo, webhookSvc, cfg.WebhookURL, cfg.WebhookSecret)

	// Build the per-user message rate limiter (0 rps = disabled).
	var msgLimiter *ratelimit.Limiter
	if cfg.RateLimitMessages > 0 {
		msgLimiter = ratelimit.New(cfg.RateLimitMessages, cfg.RateLimitMessagesBurst)
		slog.Info("Message rate limiting enabled",
			"rps", cfg.RateLimitMessages,
			"burst", cfg.RateLimitMessagesBurst)
	}

	restHandler := rest.NewHandler(chatroomSvc, messageSvc, realtimeSvc, deliverySvc, botSvc, database.DB, cfg)
	wsHandler := ws.NewHandler(chatroomSvc, messageSvc, realtimeSvc, deliverySvc, botSvc, cfg, msgLimiter)

	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", restHandler.HandleHealth)
	mux.HandleFunc("/metrics", restHandler.HandleMetrics)
	mux.HandleFunc("/ws", wsHandler.HandleConnection)

	// Protected routes — JWT required
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /rooms", restHandler.HandleGetUserRooms)
	protectedMux.HandleFunc("POST /rooms", restHandler.HandleCreateRoom)
	protectedMux.HandleFunc("GET /rooms/{room_id}", restHandler.HandleGetRoom)
	protectedMux.HandleFunc("PATCH /rooms/{room_id}", restHandler.HandleUpdateRoom)
	protectedMux.HandleFunc("GET /rooms/{room_id}/members", restHandler.HandleGetRoomMembers)
	protectedMux.HandleFunc("POST /rooms/{room_id}/members", restHandler.HandleAddMember)
	protectedMux.HandleFunc("POST /rooms/{room_id}/messages", rateLimited(msgLimiter, restHandler.HandleSendMessage))
	protectedMux.HandleFunc("GET /rooms/{room_id}/messages", restHandler.HandleGetMessages)
	protectedMux.HandleFunc("DELETE /rooms/{room_id}/messages/{message_id}", restHandler.HandleDeleteMessage)
	protectedMux.HandleFunc("PUT /rooms/{room_id}/messages/{message_id}", restHandler.HandleEditMessage)
	protectedMux.HandleFunc("POST /acks", restHandler.HandleAck)
	protectedMux.HandleFunc("GET /admin/dead-letters", restHandler.HandleGetDeadLetters)
	protectedMux.HandleFunc("POST /bots", restHandler.HandleCreateBot)
	protectedMux.HandleFunc("GET /bots", restHandler.HandleListBots)
	protectedMux.HandleFunc("GET /bots/{bot_id}", restHandler.HandleGetBot)
	protectedMux.HandleFunc("DELETE /bots/{bot_id}", restHandler.HandleDeleteBot)

	mux.Handle("/", restHandler.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		protectedMux.ServeHTTP(w, r)
	}))

	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      corsMiddleware(cfg.AllowedOrigins, mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer:  httpServer,
		config:      cfg,
		realtimeSvc: realtimeSvc,
		msgLimiter:  msgLimiter,
	}
}

// rateLimited wraps a handler with per-user rate limiting. If limiter is nil
// (rate limiting disabled) the handler is returned unchanged.
func rateLimited(limiter *ratelimit.Limiter, next http.HandlerFunc) http.HandlerFunc {
	if limiter == nil {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		if userID != "" && !limiter.Allow(userID) {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limited","message":"too many requests"}`))
			return
		}
		next(w, r)
	}
}

// corsMiddleware adds CORS headers.
func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}
	_, wildcard := originSet["*"]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		var allowOrigin string
		if wildcard {
			allowOrigin = "*"
		} else if origin != "" {
			if _, ok := originSet[origin]; ok || len(allowedOrigins) == 0 {
				allowOrigin = origin
			}
		}

		if allowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	slog.Info("Starting HTTP server", "addr", s.config.ListenAddr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	slog.Info("Shutting down HTTP server")

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownDrainTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	if err := s.realtimeSvc.Shutdown(ctx); err != nil {
		slog.Error("Realtime service shutdown error", "error", err)
	}

	if s.msgLimiter != nil {
		s.msgLimiter.Stop()
	}

	slog.Info("HTTP server shutdown complete")
}
