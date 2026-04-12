package message

import (
	"log/slog"
	"sync/atomic"

	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/repository"
)

// Service handles message operations
type Service struct {
	repo         repository.MessageRepository
	messagesSent atomic.Int64
}

// NewService creates a new message service
func NewService(repo repository.MessageRepository) *Service {
	return &Service{repo: repo}
}

// SendMessage stores a message transactionally with sequencing
func (s *Service) SendMessage(roomID, senderID string, req *models.CreateMessageRequest) (*models.Message, error) {
	msg, err := s.repo.Send(roomID, senderID, req)
	if err != nil {
		return nil, err
	}
	s.messagesSent.Add(1)
	slog.Info("Message sent", "room_id", roomID, "message_id", msg.MessageID, "sender_id", senderID, "seq", msg.Seq)
	return msg, nil
}

// MessagesSent returns the total number of messages sent since startup.
func (s *Service) MessagesSent() int64 {
	return s.messagesSent.Load()
}

// GetMessages retrieves messages for a room with pagination
func (s *Service) GetMessages(roomID string, afterSeq, limit int) ([]*models.Message, error) {
	return s.repo.List(roomID, afterSeq, limit)
}

// GetMessage retrieves a single message by ID
func (s *Service) GetMessage(messageID string) (*models.Message, error) {
	return s.repo.GetByID(messageID)
}

// GetLastAckSeq gets the last acknowledged sequence for a user in a room
func (s *Service) GetLastAckSeq(userID, roomID string) (int, error) {
	return s.repo.GetLastAckSeq(userID, roomID)
}

// UpdateLastAck updates the last acknowledged sequence for a user in a room
func (s *Service) UpdateLastAck(userID, roomID string, seq int) error {
	return s.repo.UpdateLastAck(userID, roomID, seq)
}

// QueueUndeliveredMessage queues a message for delivery to offline users
func (s *Service) QueueUndeliveredMessage(userID, roomID, messageID string, seq int) error {
	return s.repo.QueueUndelivered(userID, roomID, messageID, seq)
}

// GetUndeliveredMessages gets undelivered messages for a user
func (s *Service) GetUndeliveredMessages(userID string, limit int) ([]*models.UndeliveredMessage, error) {
	return s.repo.GetUndelivered(userID, limit)
}

// MarkMessageDelivered removes a message from the undelivered queue
func (s *Service) MarkMessageDelivered(id int) error {
	return s.repo.MarkDelivered(id)
}

// GetFailedUndeliveredMessages retrieves undelivered messages that have exceeded retry attempts
func (s *Service) GetFailedUndeliveredMessages(limit int) ([]*models.UndeliveredMessage, error) {
	return s.repo.GetFailed(limit)
}

// DeleteMessage deletes a message. Only the original sender may delete.
func (s *Service) DeleteMessage(roomID, messageID, senderID string) (int, error) {
	seq, err := s.repo.Delete(roomID, messageID, senderID)
	if err != nil {
		return 0, err
	}
	slog.Info("Message deleted", "room_id", roomID, "message_id", messageID, "sender_id", senderID, "seq", seq)
	return seq, nil
}

// UpdateMessage updates the content of a message. Only the original sender may edit.
func (s *Service) UpdateMessage(roomID, messageID, senderID, newContent string) (*models.Message, error) {
	msg, err := s.repo.Update(roomID, messageID, senderID, newContent)
	if err != nil {
		return nil, err
	}
	slog.Info("Message edited", "room_id", roomID, "message_id", messageID, "sender_id", senderID)
	return msg, nil
}
