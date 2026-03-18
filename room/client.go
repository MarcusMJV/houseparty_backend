package room

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait     = 60 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type Client struct {
	ID         string
	Name       string
	Connection *websocket.Conn
	RoomCode   string
	Manager    *RoomManager
	Egress     chan Event
}

func generateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Println("failed to generate client ID: ", err.Error())
	}
	return hex.EncodeToString(b)
}

func newClient(name string, conn *websocket.Conn, roomCode string, manager *RoomManager) *Client {
	var cleintID string
	name = manager.CheckClientName(name, roomCode, 2)
	if id, ok := manager.CheckClientHistory(roomCode, name); ok {
		cleintID = id
	} else {
		cleintID = generateClientID()
	}

	return &Client{
		ID:         cleintID,
		Name:       name,
		Connection: conn,
		RoomCode:   roomCode,
		Manager:    manager,
		Egress:     make(chan Event),
	}
}

func (c *Client) GetClientRoom() *Room {
	c.Manager.mu.Lock()
	defer c.Manager.mu.Unlock()
	return c.Manager.Rooms[c.RoomCode]
}

func (c *Client) ReadMessages() {
	defer func() {
		if _, ok := c.Manager.Rooms[c.RoomCode]; ok {
			c.Manager.RemoveClient(c)
		}
		// c.Manager.RemoveClient(c)
	}()

	err := c.Connection.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Println("failed to set read deadline: ", err.Error())
		return
	}

	c.Connection.SetReadLimit(512)
	c.Connection.SetPongHandler(c.PongHnadler)

	for {
		_, payLoad, err := c.Connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err.Error())
			}
			break
		}

		var request Event
		if err := json.Unmarshal(payLoad, &request); err != nil {
			log.Println("failed to unmarshal message: ", err.Error())
			break
		}

		if err := c.Manager.routeEvent(request, c); err != nil {
			log.Println("failed to route event: ", err.Error())
			break
		}

	}
}

func (c *Client) WriteMessages() {
	defer func() {
		if _, ok := c.Manager.Rooms[c.RoomCode]; ok {
			c.Manager.RemoveClient(c)
		}
		// c.Manager.RemoveClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.Egress:

			if !ok {
				if err := c.Connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Println("conncetion closed: ", err.Error())
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println("failed to marshal message: ", err.Error())
				return
			}

			if err := c.Connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("failed to send message: ", err.Error())
			}

		case <-ticker.C:
			if err := c.Connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("failed to send ping: ", err.Error())
				return
			}
		}
	}
}

func (c *Client) PongHnadler(pongMessage string) error {
	return c.Connection.SetReadDeadline(time.Now().Add(pongWait))
}
