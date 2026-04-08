package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hastenr/chatapi/internal/models"
)

// SQLiteMessageRepository implements repository.MessageRepository using SQLite.
type SQLiteMessageRepository struct {
	db *sql.DB
}

// NewMessageRepository creates a new SQLiteMessageRepository.
func NewMessageRepository(db *sql.DB) *SQLiteMessageRepository {
	return &SQLiteMessageRepository{db: db}
}

// Send stores a message transactionally with sequencing.
// It increments last_seq in the rooms table, reads it back, then inserts the message — all in one tx.
func (r *SQLiteMessageRepository) Send(roomID, senderID string, req *models.CreateMessageRequest) (*models.Message, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`UPDATE rooms SET last_seq = last_seq + 1 WHERE room_id = ?`, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to update room sequence: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, fmt.Errorf("room not found")
	}

	var seq int
	if err := tx.QueryRow(`SELECT last_seq FROM rooms WHERE room_id = ?`, roomID).Scan(&seq); err != nil {
		return nil, fmt.Errorf("failed to get sequence number: %w", err)
	}

	messageID := uuid.New().String()
	now := time.Now()

	_, err = tx.Exec(
		`INSERT INTO messages (message_id, chatroom_id, sender_id, seq, content, meta, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		messageID, roomID, senderID, seq, req.Content, req.Meta, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.Message{
		MessageID:  messageID,
		ChatroomID: roomID,
		SenderID:   senderID,
		Seq:        seq,
		Content:    req.Content,
		Meta:       req.Meta,
		CreatedAt:  now,
	}, nil
}

// GetByID retrieves a single message by ID.
func (r *SQLiteMessageRepository) GetByID(messageID string) (*models.Message, error) {
	var msg models.Message
	err := r.db.QueryRow(
		`SELECT message_id, chatroom_id, sender_id, seq, content, meta, created_at FROM messages WHERE message_id = ?`,
		messageID,
	).Scan(&msg.MessageID, &msg.ChatroomID, &msg.SenderID, &msg.Seq, &msg.Content, &msg.Meta, &msg.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &msg, nil
}

// List retrieves messages for a room with pagination.
func (r *SQLiteMessageRepository) List(roomID string, afterSeq, limit int) ([]*models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `SELECT message_id, chatroom_id, sender_id, seq, content, meta, created_at FROM messages WHERE chatroom_id = ?`
	args := []interface{}{roomID}

	if afterSeq > 0 {
		query += " AND seq > ?"
		args = append(args, afterSeq)
	}
	query += " ORDER BY seq ASC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.MessageID, &msg.ChatroomID, &msg.SenderID, &msg.Seq, &msg.Content, &msg.Meta, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}

// Update updates the content of a message. Only the original sender may edit.
func (r *SQLiteMessageRepository) Update(roomID, messageID, senderID, content string) (*models.Message, error) {
	var storedSenderID string
	err := r.db.QueryRow(
		`SELECT sender_id FROM messages WHERE chatroom_id = ? AND message_id = ?`,
		roomID, messageID,
	).Scan(&storedSenderID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to look up message: %w", err)
	}
	if storedSenderID != senderID {
		return nil, fmt.Errorf("forbidden")
	}

	_, err = r.db.Exec(
		`UPDATE messages SET content = ? WHERE chatroom_id = ? AND message_id = ?`,
		content, roomID, messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return r.GetByID(messageID)
}

// Delete deletes a message. Only the original sender may delete.
// Returns the deleted message's seq number.
func (r *SQLiteMessageRepository) Delete(roomID, messageID, senderID string) (int, error) {
	var storedSenderID string
	var seq int
	err := r.db.QueryRow(
		`SELECT sender_id, seq FROM messages WHERE chatroom_id = ? AND message_id = ?`,
		roomID, messageID,
	).Scan(&storedSenderID, &seq)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("message not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to look up message: %w", err)
	}
	if storedSenderID != senderID {
		return 0, fmt.Errorf("forbidden")
	}

	_, err = r.db.Exec(`DELETE FROM messages WHERE chatroom_id = ? AND message_id = ?`, roomID, messageID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete message: %w", err)
	}
	return seq, nil
}

// GetLastAckSeq gets the last acknowledged sequence for a user in a room.
func (r *SQLiteMessageRepository) GetLastAckSeq(userID, roomID string) (int, error) {
	var lastAck int
	err := r.db.QueryRow(
		`SELECT last_ack FROM delivery_state WHERE user_id = ? AND chatroom_id = ?`,
		userID, roomID,
	).Scan(&lastAck)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get last ack seq: %w", err)
	}
	return lastAck, nil
}

// UpdateLastAck updates the last acknowledged sequence for a user in a room.
func (r *SQLiteMessageRepository) UpdateLastAck(userID, roomID string, seq int) error {
	_, err := r.db.Exec(`
		INSERT INTO delivery_state (user_id, chatroom_id, last_ack, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, chatroom_id) DO UPDATE SET
			last_ack = CASE WHEN excluded.last_ack > last_ack THEN excluded.last_ack ELSE last_ack END,
			updated_at = CURRENT_TIMESTAMP
		WHERE excluded.last_ack > last_ack
	`, userID, roomID, seq)
	if err != nil {
		return fmt.Errorf("failed to update last ack: %w", err)
	}
	return nil
}

// QueueUndelivered inserts an entry into the undelivered_messages table.
func (r *SQLiteMessageRepository) QueueUndelivered(userID, roomID, messageID string, seq int) error {
	_, err := r.db.Exec(
		`INSERT INTO undelivered_messages (user_id, chatroom_id, message_id, seq) VALUES (?, ?, ?, ?)`,
		userID, roomID, messageID, seq,
	)
	if err != nil {
		return fmt.Errorf("failed to queue undelivered message: %w", err)
	}
	return nil
}

// GetUndelivered gets undelivered messages for a user.
func (r *SQLiteMessageRepository) GetUndelivered(userID string, limit int) ([]*models.UndeliveredMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, chatroom_id, message_id, seq, attempts, created_at, last_attempt_at
		FROM undelivered_messages WHERE user_id = ? ORDER BY seq ASC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get undelivered messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.UndeliveredMessage
	for rows.Next() {
		var msg models.UndeliveredMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.ChatroomID, &msg.MessageID, &msg.Seq, &msg.Attempts, &msg.CreatedAt, &msg.LastAttemptAt); err != nil {
			return nil, fmt.Errorf("failed to scan undelivered message: %w", err)
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}

// MarkDelivered removes a message from the undelivered queue.
func (r *SQLiteMessageRepository) MarkDelivered(id int) error {
	_, err := r.db.Exec(`DELETE FROM undelivered_messages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to mark message delivered: %w", err)
	}
	return nil
}

// GetFailed retrieves undelivered messages that have exceeded retry attempts.
func (r *SQLiteMessageRepository) GetFailed(limit int) ([]*models.UndeliveredMessage, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.db.Query(`
		SELECT id, user_id, chatroom_id, message_id, seq, attempts, created_at, last_attempt_at
		FROM undelivered_messages WHERE attempts >= 5 ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed undelivered messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.UndeliveredMessage
	for rows.Next() {
		var msg models.UndeliveredMessage
		if err := rows.Scan(&msg.ID, &msg.UserID, &msg.ChatroomID, &msg.MessageID, &msg.Seq, &msg.Attempts, &msg.CreatedAt, &msg.LastAttemptAt); err != nil {
			return nil, fmt.Errorf("failed to scan undelivered message: %w", err)
		}
		messages = append(messages, &msg)
	}
	return messages, rows.Err()
}
