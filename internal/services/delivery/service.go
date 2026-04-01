package delivery

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hastenr/chatapi/internal/models"
	"github.com/hastenr/chatapi/internal/services/chatroom"
	"github.com/hastenr/chatapi/internal/services/realtime"
	"github.com/hastenr/chatapi/internal/services/webhook"
)

// Service handles message and notification delivery with retries
type Service struct {
	db               *sql.DB
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
	db *sql.DB,
	realtimeSvc *realtime.Service,
	chatroomSvc *chatroom.Service,
	webhookURL string,
	webhookSecret string,
	webhookSvc *webhook.Service,
) *Service {
	return &Service{
		db:            db,
		realtimeSvc:   realtimeSvc,
		chatroomSvc:   chatroomSvc,
		webhookSvc:    webhookSvc,
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
		maxAttempts:   5,
	}
}

// ProcessUndeliveredMessages processes messages that haven't been delivered yet
func (s *Service) ProcessUndeliveredMessages(tenantID string, limit int) error {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, tenant_id, user_id, chatroom_id, message_id, seq, attempts
		FROM undelivered_messages
		WHERE tenant_id = ? AND attempts < ?
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := s.db.Query(query, tenantID, s.maxAttempts, limit)
	if err != nil {
		return fmt.Errorf("failed to get undelivered messages: %w", err)
	}

	// Collect all rows before closing the cursor. SQLite does not allow writes
	// to a table while a read cursor is open on it.
	var pending []models.UndeliveredMessage
	for rows.Next() {
		var msg models.UndeliveredMessage
		err := rows.Scan(
			&msg.ID,
			&msg.TenantID,
			&msg.UserID,
			&msg.ChatroomID,
			&msg.MessageID,
			&msg.Seq,
			&msg.Attempts,
		)
		if err != nil {
			slog.Error("Failed to scan undelivered message", "error", err)
			continue
		}
		pending = append(pending, msg)
	}
	rows.Close()

	for i := range pending {
		if err := s.attemptMessageDelivery(&pending[i]); err != nil {
			slog.Warn("Failed to deliver message",
				"tenant_id", tenantID,
				"message_id", pending[i].MessageID,
				"user_id", pending[i].UserID,
				"attempts", pending[i].Attempts,
				"error", err)
		}
	}

	return nil
}

// attemptMessageDelivery tries to deliver a message to a user
func (s *Service) attemptMessageDelivery(msg *models.UndeliveredMessage) error {
	// Check if user is online
	s.deliveryAttempts.Add(1)
	if s.realtimeSvc.IsUserOnline(msg.TenantID, msg.UserID) {
		// Get the full message to send
		fullMsg, err := s.getMessage(msg.TenantID, msg.MessageID)
		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		// Send via WebSocket
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

		s.realtimeSvc.SendToUser(msg.TenantID, msg.UserID, messagePayload)

		// Mark as delivered
		return s.markMessageDelivered(msg.ID)
	}

	// User is offline, increment attempts
	s.deliveryFailures.Add(1)
	return s.incrementMessageAttempts(msg.ID)
}

// HandleNewMessage queues undelivered messages and fires webhooks for offline room members.
// Call this after a message has been sent and broadcast to online users.
// The work runs in the calling goroutine — wrap in go if you don't want to block.
func (s *Service) HandleNewMessage(tenantID, roomID string, message *models.Message) {
	members, err := s.chatroomSvc.GetRoomMembers(tenantID, roomID)
	if err != nil {
		slog.Error("HandleNewMessage: failed to get room members",
			"tenant_id", tenantID, "room_id", roomID, "error", err)
		return
	}

	room, err := s.chatroomSvc.GetRoom(tenantID, roomID)
	if err != nil {
		slog.Error("HandleNewMessage: failed to get room",
			"tenant_id", tenantID, "room_id", roomID, "error", err)
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
		if s.realtimeSvc.IsUserOnline(tenantID, member.UserID) {
			continue
		}

		// Queue for the delivery worker to retry
		if err := s.queueUndelivered(tenantID, member.UserID, roomID, message.MessageID, message.Seq); err != nil {
			slog.Error("HandleNewMessage: failed to queue undelivered message",
				"tenant_id", tenantID,
				"user_id", member.UserID,
				"message_id", message.MessageID,
				"error", err)
		}

		// Fire webhook immediately so the app can push a notification
		if s.webhookURL != "" {
			go s.webhookSvc.NotifyOfflineUser(s.webhookURL, s.webhookSecret, tenantID, roomID, member.UserID, room.Metadata, msgInfo)
		}
	}
}

// queueUndelivered inserts an entry into the undelivered_messages table.
func (s *Service) queueUndelivered(tenantID, userID, roomID, messageID string, seq int) error {
	query := `
		INSERT INTO undelivered_messages (tenant_id, user_id, chatroom_id, message_id, seq)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, tenantID, userID, roomID, messageID, seq)
	return err
}

// DeliveryAttempts returns the total number of message delivery attempts since startup.
func (s *Service) DeliveryAttempts() int64 {
	return s.deliveryAttempts.Load()
}

// DeliveryFailures returns the number of delivery attempts where the user was offline.
func (s *Service) DeliveryFailures() int64 {
	return s.deliveryFailures.Load()
}

// ProcessNotifications processes pending notifications
func (s *Service) ProcessNotifications(tenantID string, limit int) error {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT notification_id, tenant_id, topic, payload, targets, attempts
		FROM notifications
		WHERE tenant_id = ? AND status IN ('pending', 'processing') AND attempts < ?
		ORDER BY created_at ASC LIMIT ?`,
		tenantID, s.maxAttempts, limit,
	)
	if err != nil {
		return fmt.Errorf("failed to get pending notifications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var notif models.Notification
		if err := rows.Scan(
			&notif.NotificationID,
			&notif.TenantID,
			&notif.Topic,
			&notif.Payload,
			&notif.Targets,
			&notif.Attempts,
		); err != nil {
			slog.Error("Failed to scan notification", "error", err)
			continue
		}
		if err := s.attemptNotificationDelivery(&notif); err != nil {
			slog.Warn("Failed to deliver notification",
				"tenant_id", tenantID,
				"notification_id", notif.NotificationID,
				"topic", notif.Topic,
				"attempts", notif.Attempts,
				"error", err)
		}
	}

	return nil
}

// DeliverNow immediately delivers a notification to online subscribers.
// It is called by the HTTP handler after creating a notification so that
// recipients do not have to wait for the next worker tick.
func (s *Service) DeliverNow(notif *models.Notification) {
	if err := s.attemptNotificationDelivery(notif); err != nil {
		slog.Warn("Immediate notification delivery failed",
			"notification_id", notif.NotificationID,
			"error", err)
	}
}

// attemptNotificationDelivery delivers a notification to the appropriate recipients.
// Delivery is scoped by targets: specific user IDs, room members, topic subscribers,
// or all online users in the tenant when no targets are specified.
func (s *Service) attemptNotificationDelivery(notif *models.Notification) error {
	payload := map[string]interface{}{
		"type":            "notification",
		"notification_id": notif.NotificationID,
		"topic":           notif.Topic,
		"payload":         notif.Payload,
		"timestamp":       time.Now().Unix(),
	}

	recipients := s.resolveRecipients(notif)
	for _, userID := range recipients {
		s.realtimeSvc.SendToUser(notif.TenantID, userID, payload)
	}

	return s.markNotificationDelivered(notif.NotificationID)
}

// resolveRecipients determines which online users should receive a notification.
func (s *Service) resolveRecipients(notif *models.Notification) []string {
	var targets models.NotificationTargets
	if notif.Targets != "" {
		if err := json.Unmarshal([]byte(notif.Targets), &targets); err != nil {
			slog.Warn("Failed to parse notification targets, falling back to broadcast", "error", err)
		}
	}

	seen := make(map[string]struct{})
	var recipients []string
	add := func(userID string) {
		if _, ok := seen[userID]; !ok {
			seen[userID] = struct{}{}
			recipients = append(recipients, userID)
		}
	}

	// Explicit user list
	for _, uid := range targets.UserIDs {
		add(uid)
	}

	// Room members
	if targets.RoomID != "" {
		if members, err := s.chatroomSvc.GetRoomMembers(notif.TenantID, targets.RoomID); err == nil {
			for _, m := range members {
				add(m.UserID)
			}
		}
	}

	// Topic subscribers
	if targets.TopicSubscribers {
		rows, err := s.db.Query(
			`SELECT subscriber_id FROM notification_subscriptions WHERE tenant_id = ? AND topic = ?`,
			notif.TenantID, notif.Topic,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uid string
				if rows.Scan(&uid) == nil {
					add(uid)
				}
			}
		}
	}

	// Fallback: broadcast to all online users when no targets specified
	if len(recipients) == 0 {
		return s.realtimeSvc.GetOnlineUsers(notif.TenantID)
	}

	// Filter to online users only
	online := make(map[string]struct{})
	for _, uid := range s.realtimeSvc.GetOnlineUsers(notif.TenantID) {
		online[uid] = struct{}{}
	}
	var online_recipients []string
	for _, uid := range recipients {
		if _, ok := online[uid]; ok {
			online_recipients = append(online_recipients, uid)
		}
	}
	return online_recipients
}

// CleanupOldEntries removes old delivered entries to prevent unbounded growth
func (s *Service) CleanupOldEntries(tenantID string, maxAge time.Duration) error {
	cutoffTime := time.Now().Add(-maxAge)

	// Clean up old undelivered messages that are marked as delivered
	// (In practice, you'd have a separate delivered_messages table)

	// For now, just clean up very old undelivered messages that have exceeded max attempts
	query := `
		DELETE FROM undelivered_messages
		WHERE tenant_id = ? AND attempts >= ? AND created_at < ?
	`

	_, err := s.db.Exec(query, tenantID, s.maxAttempts, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old undelivered messages: %w", err)
	}

	// Clean up old dead notifications
	notifQuery := `
		DELETE FROM notifications
		WHERE tenant_id = ? AND status = 'dead' AND created_at < ?
	`

	_, err = s.db.Exec(notifQuery, tenantID, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old notifications: %w", err)
	}

	slog.Info("Cleaned up old delivery entries",
		"tenant_id", tenantID,
		"max_age", maxAge)

	return nil
}

// Helper methods

func (s *Service) getMessage(tenantID, messageID string) (*models.Message, error) {
	var msg models.Message
	query := `
		SELECT message_id, tenant_id, chatroom_id, sender_id, seq, content, meta, created_at
		FROM messages
		WHERE tenant_id = ? AND message_id = ?
	`

	err := s.db.QueryRow(query, tenantID, messageID).Scan(
		&msg.MessageID,
		&msg.TenantID,
		&msg.ChatroomID,
		&msg.SenderID,
		&msg.Seq,
		&msg.Content,
		&msg.Meta,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func (s *Service) markMessageDelivered(id int) error {
	query := `DELETE FROM undelivered_messages WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *Service) incrementMessageAttempts(id int) error {
	query := `
		UPDATE undelivered_messages
		SET attempts = attempts + 1, last_attempt_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *Service) markNotificationDelivered(notificationID string) error {
	query := `
		UPDATE notifications
		SET status = 'delivered', last_attempt_at = CURRENT_TIMESTAMP
		WHERE notification_id = ?
	`
	_, err := s.db.Exec(query, notificationID)
	return err
}
