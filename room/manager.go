package room

import "sync"

type RoomManager struct {
	Rooms map[string]*Room
	mu    sync.RWMutex
}

func (m *RoomManager) CreateRoom() *Room {

	code := GenerateRoomCode(5)

	room := &Room{
		Code: code,
	}

	m.mu.Lock()
	m.Rooms[code] = room
	m.mu.Unlock()

	return room
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}
