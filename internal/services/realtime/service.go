package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/getchatapi/chatapi/internal/broker"
	"github.com/getchatapi/chatapi/internal/repository"
)

// Service manages WebSocket connections and real-time messaging
type Service struct {
	mu                    sync.RWMutex
	roomRepo              repository.RoomRepository
	broker                broker.Broker
	connections           map[string][]*websocket.Conn // user -> connections
	presence              map[string]time.Time         // user -> last seen
	shutdownCh            chan struct{}
	shutdownOnce          sync.Once
	maxConnectionsPerUser int
	activeConnections     atomic.Int64
}

// NewService creates a new realtime service
func NewService(roomRepo repository.RoomRepository, maxConnectionsPerUser int) *Service {
	if maxConnectionsPerUser <= 0 {
		maxConnectionsPerUser = 5
	}
	s := &Service{
		roomRepo:              roomRepo,
		connections:           make(map[string][]*websocket.Conn),
		presence:              make(map[string]time.Time),
		shutdownCh:            make(chan struct{}),
		maxConnectionsPerUser: maxConnectionsPerUser,
	}
	s.broker = broker.NewLocalBroker(s.deliverToRoom)
	go s.presenceCleanupWorker()
	return s
}

// RegisterConnection registers a new WebSocket connection for a user.
func (s *Service) RegisterConnection(userID string, conn *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := len(s.connections[userID])
	if current >= s.maxConnectionsPerUser {
		slog.Warn("Connection limit reached for user", "user_id", userID, "limit", s.maxConnectionsPerUser)
		return fmt.Errorf("connection limit of %d reached", s.maxConnectionsPerUser)
	}

	s.connections[userID] = append(s.connections[userID], conn)
	s.activeConnections.Add(1)
	s.presence[userID] = time.Now()

	slog.Info("WebSocket connection registered", "user_id", userID, "total_connections", current+1)
	return nil
}

// UnregisterConnection removes a WebSocket connection for a user
func (s *Service) UnregisterConnection(userID string, conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connections, exists := s.connections[userID]
	if !exists {
		return
	}

	for i, c := range connections {
		if c == conn {
			s.connections[userID] = append(connections[:i], connections[i+1:]...)
			s.activeConnections.Add(-1)
			break
		}
	}

	if len(s.connections[userID]) == 0 {
		delete(s.connections, userID)
		shutdownCh := s.shutdownCh
		time.AfterFunc(5*time.Second, func() {
			select {
			case <-shutdownCh:
				return // service shut down, skip broadcast
			default:
			}
			broadcast := false
			s.mu.Lock()
			if presenceTime, exists := s.presence[userID]; exists {
				if time.Since(presenceTime) >= 5*time.Second {
					delete(s.presence, userID)
					broadcast = true
				}
			}
			s.mu.Unlock()
			if broadcast {
				s.broadcastPresenceUpdate(userID, "offline")
			}
		})
	}

	slog.Info("WebSocket connection unregistered", "user_id", userID, "remaining_connections", len(s.connections[userID]))
}

// BroadcastToRoom broadcasts a message to all users in a room.
func (s *Service) BroadcastToRoom(roomID string, message interface{}) {
	payload, err := json.Marshal(message)
	if err != nil {
		slog.Error("BroadcastToRoom: failed to marshal message", "room_id", roomID, "error", err)
		return
	}
	s.broker.Broadcast(roomID, payload)
}

// deliverToRoom is called by the broker for each message.
func (s *Service) deliverToRoom(roomID string, payload []byte) {
	roomMembers, err := s.roomRepo.GetMemberIDs(roomID)
	if err != nil {
		slog.Error("deliverToRoom: failed to get room members", "room_id", roomID, "error", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, memberID := range roomMembers {
		if connections, exists := s.connections[memberID]; exists {
			for _, conn := range connections {
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					slog.Warn("Failed to broadcast message to user", "user_id", memberID, "room_id", roomID, "error", err)
				}
			}
		}
	}
}

// SendToUser sends a message directly to a specific user
func (s *Service) SendToUser(userID string, message interface{}) {
	s.mu.RLock()
	connections, exists := s.connections[userID]
	s.mu.RUnlock()

	if !exists || len(connections) == 0 {
		return
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		slog.Error("Failed to marshal message for user", "user_id", userID, "error", err)
		return
	}

	for _, conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			slog.Warn("Failed to send message to user connection", "user_id", userID, "error", err)
		}
	}
}

// IsUserOnline checks if a user has active connections
func (s *Service) IsUserOnline(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	connections, exists := s.connections[userID]
	return exists && len(connections) > 0
}

// GetOnlineUsers returns all currently online users
func (s *Service) GetOnlineUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var online []string
	for userID := range s.presence {
		if conns, exists := s.connections[userID]; exists && len(conns) > 0 {
			online = append(online, userID)
		}
	}
	return online
}

// BroadcastPresenceUpdate broadcasts a presence change
func (s *Service) BroadcastPresenceUpdate(userID, status string) {
	s.broadcastPresenceUpdate(userID, status)
}

func (s *Service) broadcastPresenceUpdate(userID, status string) {
	msg := map[string]interface{}{
		"type":      "presence.update",
		"user_id":   userID,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Failed to marshal presence message", "error", err)
		return
	}

	for _, connections := range s.connections {
		for _, conn := range connections {
			if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				slog.Warn("Failed to send presence update", "error", err)
			}
		}
	}
}

func (s *Service) presenceCleanupWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupStalePresence()
		case <-s.shutdownCh:
			return
		}
	}
}

func (s *Service) cleanupStalePresence() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	staleThreshold := 5 * time.Minute

	for userID, lastSeen := range s.presence {
		if now.Sub(lastSeen) > staleThreshold && len(s.connections[userID]) == 0 {
			delete(s.presence, userID)
		}
	}
}

// ActiveConnections returns the current number of open WebSocket connections.
func (s *Service) ActiveConnections() int64 {
	return s.activeConnections.Load()
}

// DroppedBroadcasts returns the total number of messages dropped due to a full broadcast channel.
func (s *Service) DroppedBroadcasts() int64 {
	return s.broker.DroppedCount()
}

// Shutdown gracefully shuts down the realtime service
func (s *Service) Shutdown(ctx context.Context) error {
	s.shutdownOnce.Do(func() {
		s.broker.Close()
		close(s.shutdownCh)
	})

	s.mu.Lock()
	defer s.mu.Unlock()

	shutdownMsg, _ := json.Marshal(map[string]interface{}{
		"type":               "server.shutdown",
		"reconnect_after_ms": 5000,
	})

	for userID, connections := range s.connections {
		for _, conn := range connections {
			conn.WriteMessage(websocket.TextMessage, shutdownMsg)
			conn.Close()
		}
		slog.Info("Closed connections for user", "user_id", userID, "connections_closed", len(connections))
	}

	s.connections = make(map[string][]*websocket.Conn)
	s.presence = make(map[string]time.Time)

	return nil
}
