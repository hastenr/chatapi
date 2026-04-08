package models

import "time"

// Room represents a chat room
type Room struct {
	RoomID    string    `json:"room_id" db:"room_id"`
	Type      string    `json:"type" db:"type"` // "dm", "group", "channel"
	UniqueKey string    `json:"-" db:"unique_key"` // For DMs
	Name      string    `json:"name,omitempty" db:"name"`
	LastSeq   int       `json:"last_seq" db:"last_seq"`
	Metadata  string    `json:"metadata,omitempty" db:"metadata"` // Arbitrary JSON for app-level context
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RoomMember represents a user's membership in a room
type RoomMember struct {
	ChatroomID string    `json:"chatroom_id" db:"chatroom_id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Role       string    `json:"role" db:"role"`
	JoinedAt   time.Time `json:"joined_at" db:"joined_at"`
}

// Message represents a chat message
type Message struct {
	MessageID  string    `json:"message_id" db:"message_id"`
	ChatroomID string    `json:"chatroom_id" db:"chatroom_id"`
	SenderID   string    `json:"sender_id" db:"sender_id"`
	Seq        int       `json:"seq" db:"seq"`
	Content    string    `json:"content" db:"content"`
	Meta       string    `json:"meta,omitempty" db:"meta"` // JSON metadata
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// DeliveryState tracks per-user per-room delivery state
type DeliveryState struct {
	UserID     string    `json:"user_id" db:"user_id"`
	ChatroomID string    `json:"chatroom_id" db:"chatroom_id"`
	LastAck    int       `json:"last_ack" db:"last_ack"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// UndeliveredMessage represents a message that hasn't been delivered yet
type UndeliveredMessage struct {
	ID            int        `json:"id" db:"id"`
	UserID        string     `json:"user_id" db:"user_id"`
	ChatroomID    string     `json:"chatroom_id" db:"chatroom_id"`
	MessageID     string     `json:"message_id" db:"message_id"`
	Seq           int        `json:"seq" db:"seq"`
	Attempts      int        `json:"attempts" db:"attempts"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty" db:"last_attempt_at"`
}

// Notification represents a durable notification
type Notification struct {
	NotificationID string     `json:"notification_id" db:"notification_id"`
	Topic          string     `json:"topic" db:"topic"`
	Payload        string     `json:"payload" db:"payload"` // JSON payload
	Targets        string     `json:"targets,omitempty" db:"targets"` // JSON-encoded NotificationTargets
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	Status         string     `json:"status" db:"status"` // pending, processing, delivered, failed, dead
	Attempts       int        `json:"attempts" db:"attempts"`
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty" db:"last_attempt_at"`
}

// NotificationSubscription represents a subscription to notification topics
type NotificationSubscription struct {
	ID           int       `json:"id" db:"id"`
	SubscriberID string    `json:"subscriber_id" db:"subscriber_id"`
	Topic        string    `json:"topic" db:"topic"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Bot represents a registered AI bot participant
type Bot struct {
	BotID        string    `json:"bot_id"`
	Name         string    `json:"name"`
	Mode         string    `json:"mode"`                   // "llm" | "external"
	Provider     string    `json:"provider,omitempty"`     // "openai" | "anthropic"
	BaseURL      string    `json:"base_url,omitempty"`     // override for openai-compatible endpoints
	Model        string    `json:"model,omitempty"`
	APIKey       string    `json:"-"`                      // never serialized
	SystemPrompt string    `json:"system_prompt,omitempty"`
	MaxContext   int       `json:"max_context"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateBotRequest represents a request to register a bot
type CreateBotRequest struct {
	Name         string `json:"name"`
	Mode         string `json:"mode"`          // "llm" | "external"
	Provider     string `json:"provider"`      // "openai" | "anthropic"
	BaseURL      string `json:"base_url"`      // optional, for openai-compatible endpoints
	Model        string `json:"model"`
	APIKey       string `json:"api_key"`
	SystemPrompt string `json:"system_prompt"`
	MaxContext   int    `json:"max_context"`   // 0 defaults to 20
}

// AddMemberRequest represents a request to add a member to a room
type AddMemberRequest struct {
	UserID string `json:"user_id"`
}

// UpdateRoomRequest represents a request to update a room's name or metadata.
// Pointer fields: nil means "do not change this field".
type UpdateRoomRequest struct {
	Name     *string `json:"name"`
	Metadata *string `json:"metadata"`
}

// UpdateMessageRequest represents a request to edit a message's content.
type UpdateMessageRequest struct {
	Content string `json:"content"`
}

// CreateRoomRequest represents a request to create a room
type CreateRoomRequest struct {
	Type     string   `json:"type"` // "dm", "group", "channel"
	Members  []string `json:"members"`
	Name     string   `json:"name,omitempty"`
	Metadata string   `json:"metadata,omitempty"` // Arbitrary JSON (listing_id, order_id, etc.)
}

// CreateMessageRequest represents a request to send a message
type CreateMessageRequest struct {
	Content string `json:"content"`
	Meta    string `json:"meta,omitempty"` // JSON metadata
}

// AckRequest represents an acknowledgment of message delivery
type AckRequest struct {
	RoomID string `json:"room_id"`
	Seq    int    `json:"seq"`
}

// CreateNotificationRequest represents a request to create a notification
type CreateNotificationRequest struct {
	Topic   string                 `json:"topic"`
	Payload map[string]interface{} `json:"payload"`
	Targets NotificationTargets    `json:"targets"`
}

// NotificationTargets specifies who should receive a notification
type NotificationTargets struct {
	UserIDs          []string `json:"user_ids,omitempty"`
	RoomID           string   `json:"room_id,omitempty"`
	TopicSubscribers bool     `json:"topic_subscribers,omitempty"`
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// WSMessageSend represents a send message command
type WSMessageSend struct {
	RoomID  string `json:"room_id"`
	Content string `json:"content"`
	Meta    string `json:"meta,omitempty"`
}

// WSAck represents an acknowledgment
type WSAck struct {
	RoomID string `json:"room_id"`
	Seq    int    `json:"seq"`
}

// WSTyping represents a typing indicator
type WSTyping struct {
	RoomID string `json:"room_id"`
	Action string `json:"action"` // "start" or "stop"
}
