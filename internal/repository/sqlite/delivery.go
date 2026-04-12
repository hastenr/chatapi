package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/getchatapi/chatapi/internal/models"
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

