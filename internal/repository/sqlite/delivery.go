package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hastenr/chatapi/internal/models"
)

// SQLiteDeliveryRepository implements repository.DeliveryRepository using SQLite.
type SQLiteDeliveryRepository struct {
	db *sql.DB
}

// NewDeliveryRepository creates a new SQLiteDeliveryRepository.
func NewDeliveryRepository(db *sql.DB) *SQLiteDeliveryRepository {
	return &SQLiteDeliveryRepository{db: db}
}

// GetPendingUndelivered retrieves undelivered messages with fewer than maxAttempts attempts.
// Rows are collected into a slice before returning to avoid holding an open read cursor.
func (r *SQLiteDeliveryRepository) GetPendingUndelivered(maxAttempts, limit int) ([]models.UndeliveredMessage, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, chatroom_id, message_id, seq, attempts
		FROM undelivered_messages WHERE attempts < ? ORDER BY created_at ASC LIMIT ?
	`, maxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get undelivered messages: %w", err)
	}

	var pending []models.UndeliveredMessage
	for rows.Next() {
		var msg models.UndeliveredMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.ChatroomID, &msg.MessageID, &msg.Seq, &msg.Attempts); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan undelivered message: %w", err)
		}
		pending = append(pending, msg)
	}
	rows.Close()
	return pending, nil
}

// QueueUndelivered inserts an entry into the undelivered_messages table.
func (r *SQLiteDeliveryRepository) QueueUndelivered(userID, roomID, messageID string, seq int) error {
	_, err := r.db.Exec(
		`INSERT INTO undelivered_messages (user_id, chatroom_id, message_id, seq) VALUES (?, ?, ?, ?)`,
		userID, roomID, messageID, seq,
	)
	return err
}

// MarkMessageDelivered deletes an entry from undelivered_messages.
func (r *SQLiteDeliveryRepository) MarkMessageDelivered(id int) error {
	_, err := r.db.Exec(`DELETE FROM undelivered_messages WHERE id = ?`, id)
	return err
}

// IncrementMessageAttempts increments the attempts counter and sets last_attempt_at.
func (r *SQLiteDeliveryRepository) IncrementMessageAttempts(id int) error {
	_, err := r.db.Exec(`
		UPDATE undelivered_messages SET attempts = attempts + 1, last_attempt_at = CURRENT_TIMESTAMP WHERE id = ?
	`, id)
	return err
}

// GetMessageByID retrieves a message by ID.
func (r *SQLiteDeliveryRepository) GetMessageByID(messageID string) (*models.Message, error) {
	var msg models.Message
	err := r.db.QueryRow(
		`SELECT message_id, chatroom_id, sender_id, seq, content, meta, created_at FROM messages WHERE message_id = ?`,
		messageID,
	).Scan(&msg.MessageID, &msg.ChatroomID, &msg.SenderID, &msg.Seq, &msg.Content, &msg.Meta, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// DeleteOldUndelivered deletes undelivered messages that have exceeded maxAttempts and were created before the given time.
func (r *SQLiteDeliveryRepository) DeleteOldUndelivered(maxAttempts int, before time.Time) error {
	_, err := r.db.Exec(
		`DELETE FROM undelivered_messages WHERE attempts >= ? AND created_at < ?`,
		maxAttempts, before,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup old undelivered messages: %w", err)
	}
	return nil
}

// GetPendingNotifications retrieves pending notifications with fewer than maxAttempts attempts.
func (r *SQLiteDeliveryRepository) GetPendingNotifications(maxAttempts, limit int) ([]models.Notification, error) {
	rows, err := r.db.Query(`
		SELECT notification_id, topic, payload, targets, attempts
		FROM notifications WHERE status IN ('pending', 'processing') AND attempts < ?
		ORDER BY created_at ASC LIMIT ?
	`, maxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notif models.Notification
		if err := rows.Scan(&notif.NotificationID, &notif.Topic, &notif.Payload, &notif.Targets, &notif.Attempts); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notif)
	}
	return notifications, rows.Err()
}

// MarkNotificationDelivered marks a notification as delivered.
func (r *SQLiteDeliveryRepository) MarkNotificationDelivered(notificationID string) error {
	_, err := r.db.Exec(
		`UPDATE notifications SET status = 'delivered', last_attempt_at = CURRENT_TIMESTAMP WHERE notification_id = ?`,
		notificationID,
	)
	return err
}

// DeleteOldNotifications deletes dead notifications created before the given time.
func (r *SQLiteDeliveryRepository) DeleteOldNotifications(before time.Time) error {
	_, err := r.db.Exec(
		`DELETE FROM notifications WHERE status = 'dead' AND created_at < ?`,
		before,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup old notifications: %w", err)
	}
	return nil
}

// GetTopicSubscribers returns subscriber IDs for a given topic.
func (r *SQLiteDeliveryRepository) GetTopicSubscribers(topic string) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT subscriber_id FROM notification_subscriptions WHERE topic = ?`,
		topic,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("failed to scan subscriber: %w", err)
		}
		subscribers = append(subscribers, uid)
	}
	return subscribers, rows.Err()
}
