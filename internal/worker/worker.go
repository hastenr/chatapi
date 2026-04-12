package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/getchatapi/chatapi/internal/db"
	"github.com/getchatapi/chatapi/internal/services/delivery"
)

// DeliveryWorker processes undelivered messages and notifications
type DeliveryWorker struct {
	deliverySvc *delivery.Service
	interval    time.Duration
	stopCh      chan struct{}
}

// NewDeliveryWorker creates a new delivery worker
func NewDeliveryWorker(deliverySvc *delivery.Service, interval time.Duration) *DeliveryWorker {
	return &DeliveryWorker{
		deliverySvc: deliverySvc,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the delivery worker
func (w *DeliveryWorker) Start(ctx context.Context) {
	slog.Info("Starting delivery worker", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Delivery worker stopped")
			return
		case <-w.stopCh:
			slog.Info("Delivery worker stopped")
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

// Stop stops the delivery worker
func (w *DeliveryWorker) Stop() {
	close(w.stopCh)
}

func (w *DeliveryWorker) processBatch() {
	if err := w.deliverySvc.ProcessUndeliveredMessages(50); err != nil {
		slog.Error("Failed to process undelivered messages", "error", err)
	}
	if err := w.deliverySvc.CleanupOldEntries(30 * 24 * time.Hour); err != nil {
		slog.Error("Failed to cleanup old entries", "error", err)
	}
}

// WALCheckpointWorker performs periodic WAL checkpoints
type WALCheckpointWorker struct {
	db       *db.DB
	interval time.Duration
	stopCh   chan struct{}
}

// NewWALCheckpointWorker creates a new WAL checkpoint worker
func NewWALCheckpointWorker(database *db.DB, interval time.Duration) *WALCheckpointWorker {
	return &WALCheckpointWorker{
		db:       database,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the WAL checkpoint worker
func (w *WALCheckpointWorker) Start(ctx context.Context) {
	slog.Info("Starting WAL checkpoint worker", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("WAL checkpoint worker stopped")
			return
		case <-w.stopCh:
			slog.Info("WAL checkpoint worker stopped")
			return
		case <-ticker.C:
			if err := db.CheckpointWAL(w.db); err != nil {
				slog.Error("Failed to checkpoint WAL", "error", err)
			}
		}
	}
}

// Stop stops the WAL checkpoint worker
func (w *WALCheckpointWorker) Stop() {
	close(w.stopCh)
}
