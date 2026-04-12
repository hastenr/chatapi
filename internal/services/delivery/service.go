package delivery

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/repository"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/services/realtime"
	"github.com/getchatapi/chatapi/internal/services/webhook"
)

// Service handles message and notification delivery with retries
type Service struct {
	repo             repository.DeliveryRepository
	realtimeSvc      *realtime.Service
	chatroomSvc      *chatroom.Service
	webhookSvc       *webhook.Service
	webhookURL       string
	webhookSecret    string
	maxAttempts      int
	deliveryAttempts atomic.Int64
	deliveryFailures atomic.Int64
}

// NewService creates a new delivery service
func NewService(
	repo repository.DeliveryRepository,
	realtimeSvc *realtime.Service,
	chatroomSvc *chatroom.Service,
	webhookURL string,
	webhookSecret string,
	webhookSvc *webhook.Service,
) *Service {
	return &Service{
		repo:          repo,
		realtimeSvc:   realtimeSvc,
		chatroomSvc:   chatroomSvc,
		webhookSvc:    webhookSvc,
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
		maxAttempts:   5,
	}
}

// ProcessUndeliveredMessages processes messages that haven't been delivered yet
func (s *Service) ProcessUndeliveredMessages(limit int) error {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	pending, err := s.repo.GetPendingUndelivered(s.maxAttempts, limit)
	if err != nil {
		return fmt.Errorf("failed to get undelivered messages: %w", err)
	}
	for i := range pending {
		if err := s.attemptMessageDelivery(&pending[i]); err != nil {
			slog.Warn("Failed to deliver message",
				"message_id", pending[i].MessageID,
				"user_id", pending[i].UserID,
				"attempts", pending[i].Attempts,
				"error", err)
		}
	}
	return nil
}

func (s *Service) attemptMessageDelivery(msg *models.UndeliveredMessage) error {
	s.deliveryAttempts.Add(1)
	if s.realtimeSvc.IsUserOnline(msg.UserID) {
		fullMsg, err := s.repo.GetMessageByID(msg.MessageID)
		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		messagePayload := map[string]interface{}{
			"type":       "message",
			"room_id":    msg.ChatroomID,
			"seq":        msg.Seq,
			"message_id": msg.MessageID,
			"sender_id":  fullMsg.SenderID,
			"content":    fullMsg.Content,
			"created_at": fullMsg.CreatedAt.Format(time.RFC3339),
		}
		if fullMsg.Meta != "" {
			messagePayload["meta"] = fullMsg.Meta
		}

		s.realtimeSvc.SendToUser(msg.UserID, messagePayload)
		return s.repo.MarkMessageDelivered(msg.ID)
	}

	s.deliveryFailures.Add(1)
	return s.repo.IncrementMessageAttempts(msg.ID)
}

// HandleNewMessage queues undelivered messages and fires webhooks for offline room members.
func (s *Service) HandleNewMessage(roomID string, message *models.Message) {
	members, err := s.chatroomSvc.GetRoomMembers(roomID)
	if err != nil {
		slog.Error("HandleNewMessage: failed to get room members", "room_id", roomID, "error", err)
		return
	}

	room, err := s.chatroomSvc.GetRoom(roomID)
	if err != nil {
		slog.Error("HandleNewMessage: failed to get room", "room_id", roomID, "error", err)
		return
	}

	msgInfo := webhook.MessageInfo{
		MessageID: message.MessageID,
		SenderID:  message.SenderID,
		Content:   message.Content,
		Seq:       message.Seq,
		CreatedAt: message.CreatedAt,
	}

	for _, member := range members {
		if member.UserID == message.SenderID {
			continue
		}
		if s.realtimeSvc.IsUserOnline(member.UserID) {
			continue
		}

		if err := s.repo.QueueUndelivered(member.UserID, roomID, message.MessageID, message.Seq); err != nil {
			slog.Error("HandleNewMessage: failed to queue undelivered message",
				"user_id", member.UserID,
				"message_id", message.MessageID,
				"error", err)
		}

		if s.webhookURL != "" {
			go s.webhookSvc.NotifyOfflineUser(s.webhookURL, s.webhookSecret, roomID, member.UserID, room.Metadata, msgInfo)
		}
	}
}

// DeliveryAttempts returns the total number of message delivery attempts since startup.
func (s *Service) DeliveryAttempts() int64 {
	return s.deliveryAttempts.Load()
}

// DeliveryFailures returns the number of delivery attempts where the user was offline.
func (s *Service) DeliveryFailures() int64 {
	return s.deliveryFailures.Load()
}

// CleanupOldEntries removes old delivered entries to prevent unbounded growth
func (s *Service) CleanupOldEntries(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	if err := s.repo.DeleteOldUndelivered(s.maxAttempts, cutoff); err != nil {
		return err
	}
	slog.Info("Cleaned up old delivery entries", "max_age", maxAge)
	return nil
}
