package chatroom_test

import (
	"testing"

	"github.com/getchatapi/chatapi/internal/models"
	"github.com/getchatapi/chatapi/internal/repository/sqlite"
	"github.com/getchatapi/chatapi/internal/services/chatroom"
	"github.com/getchatapi/chatapi/internal/testutil"
)

func newSvc(t *testing.T) *chatroom.Service {
	t.Helper()
	db := testutil.NewTestDB(t)
	return chatroom.NewService(sqlite.NewRoomRepository(db.DB))
}

// --- CreateRoom ---

func TestCreateRoom_Group(t *testing.T) {
	svc := newSvc(t)

	room, err := svc.CreateRoom(&models.CreateRoomRequest{
		Type:    "group",
		Name:    "general",
		Members: []string{"alice", "bob"},
	})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if room.RoomID == "" {
		t.Error("room_id is empty")
	}
	if room.Type != "group" {
		t.Errorf("type = %q, want %q", room.Type, "group")
	}
	if room.Name != "general" {
		t.Errorf("name = %q, want %q", room.Name, "general")
	}
}

func TestCreateRoom_DM_Deduplication(t *testing.T) {
	svc := newSvc(t)

	req := &models.CreateRoomRequest{
		Type:    "dm",
		Members: []string{"alice", "bob"},
	}

	r1, err := svc.CreateRoom(req)
	if err != nil {
		t.Fatalf("first CreateRoom: %v", err)
	}

	// Creating a DM with the same two users must return the existing room
	r2, err := svc.CreateRoom(req)
	if err != nil {
		t.Fatalf("second CreateRoom: %v", err)
	}
	if r1.RoomID != r2.RoomID {
		t.Errorf("room_id changed on second DM creation: %q vs %q", r1.RoomID, r2.RoomID)
	}
}

func TestCreateRoom_DM_MemberOrderDoesNotMatter(t *testing.T) {
	svc := newSvc(t)

	r1, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type:    "dm",
		Members: []string{"alice", "bob"},
	})
	r2, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type:    "dm",
		Members: []string{"bob", "alice"},
	})

	if r1.RoomID != r2.RoomID {
		t.Error("DM deduplication failed: different member order produced different rooms")
	}
}

func TestCreateRoom_InvalidType(t *testing.T) {
	svc := newSvc(t)

	_, err := svc.CreateRoom(&models.CreateRoomRequest{
		Type:    "invalid",
		Members: []string{"alice", "bob"},
	})
	if err == nil {
		t.Error("expected error for invalid room type, got nil")
	}
}

func TestCreateRoom_DM_RequiresExactlyTwoMembers(t *testing.T) {
	svc := newSvc(t)

	_, err := svc.CreateRoom(&models.CreateRoomRequest{
		Type:    "dm",
		Members: []string{"alice"},
	})
	if err == nil {
		t.Error("expected error for DM with 1 member, got nil")
	}
}

func TestCreateRoom_WithMetadata(t *testing.T) {
	svc := newSvc(t)

	room, err := svc.CreateRoom(&models.CreateRoomRequest{
		Type:     "group",
		Name:     "support",
		Members:  []string{"agent", "customer"},
		Metadata: `{"ticket_id":"t_99"}`,
	})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if room.Metadata != `{"ticket_id":"t_99"}` {
		t.Errorf("metadata = %q, want %q", room.Metadata, `{"ticket_id":"t_99"}`)
	}
}

// --- GetRoom ---

func TestGetRoom_Found(t *testing.T) {
	svc := newSvc(t)

	created, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"a", "b"},
	})

	got, err := svc.GetRoom(created.RoomID)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if got.RoomID != created.RoomID {
		t.Errorf("room_id = %q, want %q", got.RoomID, created.RoomID)
	}
}

func TestGetRoom_NotFound(t *testing.T) {
	svc := newSvc(t)

	_, err := svc.GetRoom("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent room, got nil")
	}
}

func TestGetRoom_WrongID(t *testing.T) {
	svc := newSvc(t)

	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"a", "b"},
	})

	_, err := svc.GetRoom(room.RoomID + "-wrong")
	if err == nil {
		t.Error("expected error for wrong room ID, got nil")
	}
}

// --- UpdateRoom ---

func TestUpdateRoom_Name(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "old", Members: []string{"a", "b"},
	})

	newName := "new-name"
	updated, err := svc.UpdateRoom(room.RoomID, &models.UpdateRoomRequest{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateRoom: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("name = %q, want %q", updated.Name, "new-name")
	}
}

func TestUpdateRoom_Metadata(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"a", "b"},
	})

	meta := `{"order_id":"ord_1"}`
	updated, err := svc.UpdateRoom(room.RoomID, &models.UpdateRoomRequest{Metadata: &meta})
	if err != nil {
		t.Fatalf("UpdateRoom: %v", err)
	}
	if updated.Metadata != meta {
		t.Errorf("metadata = %q, want %q", updated.Metadata, meta)
	}
}

func TestUpdateRoom_NoFieldsIsNoop(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "original", Members: []string{"a", "b"},
	})

	// Empty update: both fields nil
	got, err := svc.UpdateRoom(room.RoomID, &models.UpdateRoomRequest{})
	if err != nil {
		t.Fatalf("UpdateRoom with no fields: %v", err)
	}
	if got.Name != "original" {
		t.Errorf("name changed unexpectedly: %q", got.Name)
	}
}

func TestUpdateRoom_NotFound(t *testing.T) {
	svc := newSvc(t)

	name := "x"
	_, err := svc.UpdateRoom("nonexistent", &models.UpdateRoomRequest{Name: &name})
	if err == nil {
		t.Error("expected error for nonexistent room, got nil")
	}
}

// --- Members ---

func TestGetRoomMembers(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"alice", "bob"},
	})

	members, err := svc.GetRoomMembers(room.RoomID)
	if err != nil {
		t.Fatalf("GetRoomMembers: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("got %d members, want 2", len(members))
	}
}

func TestAddMember(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"alice", "bob"},
	})

	if err := svc.AddMember(room.RoomID, "charlie"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	members, _ := svc.GetRoomMembers(room.RoomID)
	if len(members) != 3 {
		t.Errorf("got %d members after add, want 3", len(members))
	}
}

func TestRemoveMember(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"alice", "bob"},
	})

	if err := svc.RemoveMember(room.RoomID, "bob"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	members, _ := svc.GetRoomMembers(room.RoomID)
	if len(members) != 1 {
		t.Errorf("got %d members after remove, want 1", len(members))
	}
}

func TestRemoveMember_NotFound(t *testing.T) {
	svc := newSvc(t)
	room, _ := svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "test", Members: []string{"alice", "bob"},
	})

	err := svc.RemoveMember(room.RoomID, "nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent member, got nil")
	}
}

// --- GetUserRooms ---

func TestGetUserRooms(t *testing.T) {
	svc := newSvc(t)

	svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "room1", Members: []string{"alice", "bob"},
	})
	svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "room2", Members: []string{"alice", "charlie"},
	})
	svc.CreateRoom(&models.CreateRoomRequest{
		Type: "group", Name: "room3", Members: []string{"bob", "charlie"},
	})

	aliceRooms, err := svc.GetUserRooms("alice")
	if err != nil {
		t.Fatalf("GetUserRooms: %v", err)
	}
	if len(aliceRooms) != 2 {
		t.Errorf("alice has %d rooms, want 2", len(aliceRooms))
	}
}
