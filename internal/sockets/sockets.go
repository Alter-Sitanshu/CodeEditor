package sockets

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Connection struct {
	RoomID int64
	UserID int64
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	// Map of roomID -> map of userID -> Connection
	Rooms   map[int64]map[int64]*Connection
	RoomMux sync.Mutex
	// Channels for hub operations
	Register   chan *Connection
	Unregister chan *Connection
	Broadcast  chan BroadcastMessage
}

type BroadcastMessage struct {
	RoomID  int64
	Message []byte
	Sender  int64 // Not to send back to sender
}

type WSMessage struct {
	RoomID    int64 `json:"room_id"`
	UserID    int64 `json:"user_id"`
	Timestamp int64 `json:"timestamp"`
	Data      any   `json:"data"`
}

type CodeChangeData struct {
	Content string     `json:"content"`
	From    CursorData `json:"from"`
	To      CursorData `json:"to"`
}

type CursorData struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func (h *Hub) ReadMessages(c *Connection) {
	defer func() {
		h.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		// Add user info and timestamp
		msg.UserID = c.UserID
		msg.RoomID = c.RoomID
		msg.Timestamp = time.Now().Unix()

		// Broadcast to other users in room
		broadcastMsg, _ := json.Marshal(msg)
		h.Broadcast <- BroadcastMessage{
			RoomID:  c.RoomID,
			Message: broadcastMsg,
			Sender:  c.UserID,
		}

	}
}

func (h *Hub) WriteMessages(c *Connection) {
	defer func() {
		h.Unregister <- c
		c.Conn.Close()
	}()
	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[int64]map[int64]*Connection),
		Register:   make(chan *Connection),
		Unregister: make(chan *Connection),
		Broadcast:  make(chan BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {

		select {
		case conn := <-h.Register:
			// New connection to room
			h.RoomMux.Lock()
			if h.Rooms[conn.RoomID] == nil {
				h.Rooms[conn.RoomID] = make(map[int64]*Connection)
			}
			h.Rooms[conn.RoomID][conn.UserID] = conn
			h.RoomMux.Unlock()
			log.Printf("User %d joined room %d", conn.UserID, conn.RoomID)

		case conn := <-h.Unregister:
			// Remove connection from room
			h.RoomMux.Lock()
			if room, exists := h.Rooms[conn.RoomID]; exists {
				if _, exists := room[conn.UserID]; exists {
					delete(room, conn.UserID)
					close(conn.Send)
					log.Printf("User %d left room %d", conn.UserID, conn.RoomID)
				}
			}
			h.RoomMux.Unlock()

		case msg := <-h.Broadcast:
			h.RoomMux.Lock()
			// Sending message to all users in room except sender
			if room, exists := h.Rooms[msg.RoomID]; exists {
				for userID, conn := range room {
					if userID != msg.Sender {
						select {
						case conn.Send <- msg.Message:
						default:
							// Connection is dead
							delete(room, userID)
							close(conn.Send)
						}
					}
				}
			}
			h.RoomMux.Unlock()
		}
	}
}
