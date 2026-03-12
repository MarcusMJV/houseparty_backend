package room

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	websockertUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

type RoomManager struct {
	Rooms    map[string]*Room
	mu       sync.RWMutex
	Handlers map[string]EventHandler
}

func (m *RoomManager) CreateRoom() *Room {

	code := GenerateRoomCode(5)

	room := &Room{
		Code:    code,
		Clients: make(map[*Client]bool),
	}

	m.mu.Lock()
	m.Rooms[code] = room
	m.mu.Unlock()

	return room
}

func (m *RoomManager) AddClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Rooms[client.RoomCode].Clients[client] = true
}

func (m *RoomManager) RemoveClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.Rooms[client.RoomCode]

	if _, ok := room.Clients[client]; ok {
		close(client.Egress)
		client.Connection.Close()
		delete(m.Rooms[client.RoomCode].Clients, client)
	}

}

func NewRoomManager() *RoomManager {
	m := &RoomManager{
		Rooms:    make(map[string]*Room),
		Handlers: make(map[string]EventHandler),
	}

	m.SetupEventHandlers()
	return m
}

func (m *RoomManager) SetupEventHandlers() {
	m.Handlers["join-room"] = JoinRoomEvent
	m.Handlers["search-song"] = SearchSongEvent
	m.Handlers["add-song"] = AddSong
}

func (m *RoomManager) ServeWS() gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := websockertUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		client := newClient(c.Param("name"), conn, c.Param("room_code"), m)
		m.AddClient(client)
		go client.ReadMessages()
		go client.WriteMessages()
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "https://localhost:5173":
		return true
	case "https://hp-frontend.up.railway.app":
		return true
	default:
		return false
	}
}

func (m *RoomManager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.Handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
	} else {
		return errors.New("no handler for event type")
	}
	return nil
}
