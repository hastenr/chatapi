package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/getchatapi/chatapi/internal/config"
	"github.com/getchatapi/chatapi/internal/db"
	"github.com/getchatapi/chatapi/internal/repository/sqlite"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/services/delivery"
	"github.com/getchatapi/chatapi/internal/services/realtime"
	"github.com/getchatapi/chatapi/internal/services/webhook"
	"github.com/getchatapi/chatapi/internal/transport"
	"github.com/getchatapi/chatapi/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	database, err := db.New(cfg.DatabaseDSN)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	roomRepo := sqlite.NewRoomRepository(database.DB)
	delivRepo := sqlite.NewDeliveryRepository(database.DB)

	realtimeSvc := realtime.NewService(roomRepo, cfg.MaxConnectionsPerUser)
	chatroomSvc := chatroom.NewService(roomRepo)
	webhookSvc := webhook.NewService()
	deliverySvc := delivery.NewService(delivRepo, realtimeSvc, chatroomSvc, cfg.WebhookURL, cfg.WebhookSecret, webhookSvc)

	deliveryWorker := worker.NewDeliveryWorker(deliverySvc, cfg.WorkerInterval)
	walWorker := worker.NewWALCheckpointWorker(database, 5*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go deliveryWorker.Start(ctx)
	go walWorker.Start(ctx)

	server := transport.NewServer(cfg, database, realtimeSvc)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Starting ChatAPI server", "addr", cfg.ListenAddr)
		if err := server.Start(); err != nil {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdown
	slog.Info("Received shutdown signal, initiating graceful shutdown")

	server.Shutdown()
	cancel()

	time.Sleep(cfg.ShutdownDrainTimeout)
	slog.Info("ChatAPI shutdown complete")
}
