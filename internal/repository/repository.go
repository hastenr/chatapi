package repository

import (
	"time"

	"github.com/hastenr/chatapi/internal/models"
)

// RoomRepository handles rooms and membership.
type RoomRepository interface {
	GetByID(roomID string) (*models.Room, error)
	GetByUniqueKey(uniqueKey string) (*models.Room, error)
	Create(room *models.Room) error
	Update(roomID string, req *models.UpdateRoomRequest) error
	GetUserRooms(userID string) ([]*models.Room, error)
	AddMember(roomID, userID string) error
	AddMembers(roomID string, userIDs []string) error
	RemoveMember(roomID, userID string) error
	GetMembers(roomID string) ([]*models.RoomMember, error)
	GetMemberIDs(roomID string) ([]string, error)
}

// MessageRepository handles messages and per-user delivery state.
type MessageRepository interface {
	Send(roomID, senderID string, req *models.CreateMessageRequest) (*models.Message, error)
	GetByID(messageID string) (*models.Message, error)
	List(roomID string, afterSeq, limit int) ([]*models.Message, error)
	Update(roomID, messageID, senderID, content string) (*models.Message, error)
	Delete(roomID, messageID, senderID string) (int, error)
	GetLastAckSeq(userID, roomID string) (int, error)
	UpdateLastAck(userID, roomID string, seq int) error
	QueueUndelivered(userID, roomID, messageID string, seq int) error
	GetUndelivered(userID string, limit int) ([]*models.UndeliveredMessage, error)
	MarkDelivered(id int) error
	GetFailed(limit int) ([]*models.UndeliveredMessage, error)
}

// DeliveryRepository handles the delivery worker's DB operations.
type DeliveryRepository interface {
	GetPendingUndelivered(maxAttempts, limit int) ([]models.UndeliveredMessage, error)
	QueueUndelivered(userID, roomID, messageID string, seq int) error
	MarkMessageDelivered(id int) error
	IncrementMessageAttempts(id int) error
	GetMessageByID(messageID string) (*models.Message, error)
	DeleteOldUndelivered(maxAttempts int, before time.Time) error
	GetPendingNotifications(maxAttempts, limit int) ([]models.Notification, error)
	MarkNotificationDelivered(notificationID string) error
	DeleteOldNotifications(before time.Time) error
	GetTopicSubscribers(topic string) ([]string, error)
}

// NotificationRepository handles notifications and subscriptions.
type NotificationRepository interface {
	Create(req *models.CreateNotificationRequest) (*models.Notification, error)
	GetByID(notificationID string) (*models.Notification, error)
	GetFailed(limit int) ([]*models.Notification, error)
	Subscribe(subscriberID, topic string) (*models.NotificationSubscription, error)
	Unsubscribe(subscriberID string, id int) error
	ListSubscriptions(subscriberID string) ([]*models.NotificationSubscription, error)
}

// BotRepository handles bot registration.
type BotRepository interface {
	Create(req *models.CreateBotRequest) (*models.Bot, error)
	GetByID(botID string) (*models.Bot, error)
	List() ([]*models.Bot, error)
	Delete(botID string) error
	Exists(botID string) (bool, error)
}
