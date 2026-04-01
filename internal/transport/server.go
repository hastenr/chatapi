package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/hastenr/chatapi/internal/config"
	"github.com/hastenr/chatapi/internal/db"
	"github.com/hastenr/chatapi/internal/handlers/rest"
	"github.com/hastenr/chatapi/internal/handlers/ws"
	"github.com/hastenr/chatapi/internal/services/chatroom"
	"github.com/hastenr/chatapi/internal/services/delivery"
	"github.com/hastenr/chatapi/internal/services/message"
	"github.com/hastenr/chatapi/internal/services/notification"
	"github.com/hastenr/chatapi/internal/services/realtime"
	"github.com/hastenr/chatapi/internal/services/webhook"
)

// Server represents the HTTP server
type Server struct {
	httpServer  *http.Server
	config      *config.Config
	realtimeSvc *realtime.Service
}

// NewServer creates and wires up the HTTP server
func NewServer(cfg *config.Config, db *db.DB, realtimeSvc *realtime.Service) *Server {
	chatroomSvc := chatroom.NewService(db.DB)
	messageSvc := message.NewService(db.DB)
	notifSvc := notification.NewService(db.DB)
	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(db.DB, realtimeSvc, chatroomSvc, cfg.WebhookURL, cfg.WebhookSecret, webhookSvc)

	restHandler := rest.NewHandler(chatroomSvc, messageSvc, realtimeSvc, deliverySvc, notifSvc, cfg)
	wsHandler := ws.NewHandler(chatroomSvc, messageSvc, realtimeSvc, deliverySvc, cfg)

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
	protectedMux.HandleFunc("POST /rooms/{room_id}/messages", restHandler.HandleSendMessage)
	protectedMux.HandleFunc("GET /rooms/{room_id}/messages", restHandler.HandleGetMessages)
	protectedMux.HandleFunc("DELETE /rooms/{room_id}/messages/{message_id}", restHandler.HandleDeleteMessage)
	protectedMux.HandleFunc("PUT /rooms/{room_id}/messages/{message_id}", restHandler.HandleEditMessage)
	protectedMux.HandleFunc("POST /acks", restHandler.HandleAck)
	protectedMux.HandleFunc("POST /notify", restHandler.HandleNotify)
	protectedMux.HandleFunc("POST /subscriptions", restHandler.HandleSubscribe)
	protectedMux.HandleFunc("GET /subscriptions", restHandler.HandleListSubscriptions)
	protectedMux.HandleFunc("DELETE /subscriptions/{id}", restHandler.HandleUnsubscribe)
	protectedMux.HandleFunc("GET /admin/dead-letters", restHandler.HandleGetDeadLetters)

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

	slog.Info("HTTP server shutdown complete")
}
