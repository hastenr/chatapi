package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hastenr/chatapi/internal/models"
)

// SQLiteNotificationRepository implements repository.NotificationRepository using SQLite.
type SQLiteNotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new SQLiteNotificationRepository.
func NewNotificationRepository(db *sql.DB) *SQLiteNotificationRepository {
	return &SQLiteNotificationRepository{db: db}
}

// Create creates a new durable notification.
func (r *SQLiteNotificationRepository) Create(req *models.CreateNotificationRequest) (*models.Notification, error) {
	notificationID := uuid.New().String()

	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	targetsJSON, err := json.Marshal(req.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal targets: %w", err)
	}

	_, err = r.db.Exec(
		`INSERT INTO notifications (notification_id, topic, payload, targets, status) VALUES (?, ?, ?, ?, 'pending')`,
		notificationID, req.Topic, string(payloadJSON), string(targetsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return &models.Notification{
		NotificationID: notificationID,
		Topic:          req.Topic,
		Payload:        string(payloadJSON),
		Targets:        string(targetsJSON),
		Status:         "pending",
		CreatedAt:      time.Now(),
	}, nil
}

// GetByID retrieves a notification by ID.
func (r *SQLiteNotificationRepository) GetByID(notificationID string) (*models.Notification, error) {
	var n models.Notification
	err := r.db.QueryRow(`
		SELECT notification_id, topic, payload, created_at, status, attempts, last_attempt_at
		FROM notifications WHERE notification_id = ?`, notificationID,
	).Scan(&n.NotificationID, &n.Topic, &n.Payload, &n.CreatedAt, &n.Status, &n.Attempts, &n.LastAttemptAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	return &n, nil
}

// GetFailed retrieves notifications that have failed delivery (status = 'dead').
func (r *SQLiteNotificationRepository) GetFailed(limit int) ([]*models.Notification, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.db.Query(`
		SELECT notification_id, topic, payload, created_at, status, attempts, last_attempt_at
		FROM notifications WHERE status = 'dead' ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.NotificationID, &n.Topic, &n.Payload, &n.CreatedAt, &n.Status, &n.Attempts, &n.LastAttemptAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &n)
	}
	return notifications, rows.Err()
}

// Subscribe subscribes a user to a notification topic.
func (r *SQLiteNotificationRepository) Subscribe(subscriberID, topic string) (*models.NotificationSubscription, error) {
	result, err := r.db.Exec(
		`INSERT INTO notification_subscriptions (subscriber_id, topic) VALUES (?, ?)`,
		subscriberID, topic,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	id, _ := result.LastInsertId()
	return &models.NotificationSubscription{
		ID:           int(id),
		SubscriberID: subscriberID,
		Topic:        topic,
		CreatedAt:    time.Now(),
	}, nil
}

// Unsubscribe removes a subscription owned by the given subscriber.
func (r *SQLiteNotificationRepository) Unsubscribe(subscriberID string, id int) error {
	result, err := r.db.Exec(
		`DELETE FROM notification_subscriptions WHERE id = ? AND subscriber_id = ?`,
		id, subscriberID,
	)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

// ListSubscriptions returns all subscriptions for a user.
func (r *SQLiteNotificationRepository) ListSubscriptions(subscriberID string) ([]*models.NotificationSubscription, error) {
	rows, err := r.db.Query(`
		SELECT id, subscriber_id, topic, created_at
		FROM notification_subscriptions WHERE subscriber_id = ? ORDER BY created_at DESC
	`, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*models.NotificationSubscription
	for rows.Next() {
		var sub models.NotificationSubscription
		if err := rows.Scan(&sub.ID, &sub.SubscriberID, &sub.Topic, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}
	return subs, rows.Err()
}
