package room

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

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
		Code:            code,
		Clients:         make(map[*Client]bool),
		ClientHistory:   make(map[string]string),
		ReconnectTimers: make(map[string]*time.Timer),
	}

	m.mu.Lock()
	m.Rooms[code] = room
	m.mu.Unlock()

	return room
}

func (m *RoomManager) AddClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := m.Rooms[client.RoomCode]

	// Cancel any pending reconnect timer for this client ID
	if timer, ok := room.ReconnectTimers[client.ID]; ok {
		timer.Stop()
		delete(room.ReconnectTimers, client.ID)
	}

	if len(room.Clients) == 0 {
		room.HostID = client.ID
	}

	room.Clients[client] = true
}

func (m *RoomManager) RemoveClient(client *Client) {
	m.mu.Lock()
	room := m.Rooms[client.RoomCode]

	if _, ok := room.Clients[client]; !ok {
		m.mu.Unlock()
		return
	}

	close(client.Egress)
	client.Connection.Close()
	room.ClientHistory[client.Name] = client.ID
	delete(room.Clients, client)

	clientID := client.ID
	clientName := client.Name

	timer := time.AfterFunc(30*time.Second, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		// If client reconnected, ClientHistory entry was removed — do nothing
		if _, pending := room.ClientHistory[clientName]; !pending {
			return
		}

		delete(room.ClientHistory, clientName)
		delete(room.ReconnectTimers, clientID)

		payload, err := json.Marshal(UserLeftPayload{Name: clientName, ID: clientID})
		if err != nil {
			return
		}
		event := Event{Type: "user_left", Payload: payload}
		for c := range room.Clients {
			c.Egress <- event
		}
	})
	room.ReconnectTimers[clientID] = timer

	m.mu.Unlock()
}

func (m *RoomManager) CheckClientHistory(roomCode string, key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := m.Rooms[roomCode]
	if id, ok := room.ClientHistory[key]; ok {
		defer delete(room.ClientHistory, key)
		return id, true
	}

	return "", false
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
	m.Handlers["song-ended"] = SongEnded
	m.Handlers["song-skip-vote"] = VoteToSkipSong
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
