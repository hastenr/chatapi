package chatroom

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/hastenr/chatapi/internal/models"
	"github.com/hastenr/chatapi/internal/repository"
)

// Service handles chatroom operations
type Service struct {
	repo repository.RoomRepository
}

// NewService creates a new chatroom service
func NewService(repo repository.RoomRepository) *Service {
	return &Service{repo: repo}
}

// CreateRoom creates a new chatroom
func (s *Service) CreateRoom(req *models.CreateRoomRequest) (*models.Room, error) {
	var room *models.Room
	var err error

	switch req.Type {
	case "dm":
		room, err = s.createDMRoom(req)
	case "group", "channel":
		room, err = s.createGroupRoom(req)
	default:
		return nil, fmt.Errorf("invalid room type: %s", req.Type)
	}

	if err != nil {
		return nil, err
	}

	if err := s.repo.AddMembers(room.RoomID, req.Members); err != nil {
		return nil, fmt.Errorf("failed to add members: %w", err)
	}

	slog.Info("Created room", "room_id", room.RoomID, "type", req.Type)
	return room, nil
}

func (s *Service) createDMRoom(req *models.CreateRoomRequest) (*models.Room, error) {
	if len(req.Members) != 2 {
		return nil, fmt.Errorf("DM rooms must have exactly 2 members")
	}

	members := make([]string, len(req.Members))
	copy(members, req.Members)
	sort.Strings(members)
	uniqueKey := fmt.Sprintf("dm:%s:%s", members[0], members[1])

	existing, err := s.repo.GetByUniqueKey(uniqueKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing DM: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	room := &models.Room{
		RoomID:    generateRoomID(),
		Type:      "dm",
		UniqueKey: uniqueKey,
		Name:      req.Name,
		Metadata:  req.Metadata,
	}

	if err := s.repo.Create(room); err != nil {
		return nil, fmt.Errorf("failed to create DM room: %w", err)
	}
	return room, nil
}

func (s *Service) createGroupRoom(req *models.CreateRoomRequest) (*models.Room, error) {
	if len(req.Members) < 2 {
		return nil, fmt.Errorf("group/channel rooms must have at least 2 members")
	}

	room := &models.Room{
		RoomID:   generateRoomID(),
		Type:     req.Type,
		Name:     req.Name,
		Metadata: req.Metadata,
	}

	if err := s.repo.Create(room); err != nil {
		return nil, fmt.Errorf("failed to create group room: %w", err)
	}
	return room, nil
}

// GetRoom retrieves a room by ID
func (s *Service) GetRoom(roomID string) (*models.Room, error) {
	return s.repo.GetByID(roomID)
}

// GetRoomMembers retrieves all members of a room
func (s *Service) GetRoomMembers(roomID string) ([]*models.RoomMember, error) {
	return s.repo.GetMembers(roomID)
}

// AddMember adds a single member to a room
func (s *Service) AddMember(roomID, userID string) error {
	if err := s.repo.AddMember(roomID, userID); err != nil {
		return err
	}
	slog.Info("Added member to room", "room_id", roomID, "user_id", userID)
	return nil
}

// RemoveMember removes a member from a room
func (s *Service) RemoveMember(roomID, userID string) error {
	if err := s.repo.RemoveMember(roomID, userID); err != nil {
		return err
	}
	slog.Info("Removed member from room", "room_id", roomID, "user_id", userID)
	return nil
}

// GetUserRooms returns all rooms that a user is a member of
func (s *Service) GetUserRooms(userID string) ([]*models.Room, error) {
	return s.repo.GetUserRooms(userID)
}

// UpdateRoom updates a room's name and/or metadata.
func (s *Service) UpdateRoom(roomID string, req *models.UpdateRoomRequest) (*models.Room, error) {
	if req.Name == nil && req.Metadata == nil {
		return s.GetRoom(roomID)
	}
	if err := s.repo.Update(roomID, req); err != nil {
		return nil, err
	}
	slog.Info("Updated room", "room_id", roomID)
	return s.GetRoom(roomID)
}

func generateRoomID() string {
	return uuid.New().String()
}
