package sockets

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const pongWait = 60 * time.Second
const pingPeriod = (pongWait * 9) / 10

type Connection struct {
	RoomID int64
	UserID int64
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	// Map of roomID -> map of userID -> Connection
	Rooms map[int64]map[int64]*Connection
	mutex sync.RWMutex
	// Channels for hub operations
	Register   chan *Connection
	Unregister chan *Connection
	Broadcast  chan BroadcastMessage
	ChatCast   chan ChatBroadcast
}

type BroadcastMessage struct {
	RoomID  int64
	Message []byte
	Sender  int64 // Not to send back to sender
}

type ChatBroadcast struct {
	RoomID  int64
	Message []byte
	Sender  int64 // Not to send back to sender
}

type WSMessage struct {
	RoomID    int64  `json:"room_id"`
	UserID    int64  `json:"user_id"`
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data"`
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

func (h *Hub) ReadMessagesWithVoice(c *Connection, vcm *VoiceChatManager) {
	defer func() {
		// Clean up voice chat when connection closes
		vcm.LeaveVoiceChat(c.RoomID, c.UserID)
		h.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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

		// Handle voice-related messages
		if msg.Type == "voice-join" || msg.Type == "voice-leave" ||
			msg.Type == "voice-state-update" || msg.Type == "voice-participants" {
			h.HandleVoiceMessage(c, msg, vcm)
			continue
		}

		// Handle existing message types
		broadcastMsg, _ := json.Marshal(msg)
		if msg.Type == "chat" {
			h.ChatCast <- ChatBroadcast{
				RoomID:  c.RoomID,
				Message: broadcastMsg,
				Sender:  c.UserID,
			}
		} else if msg.Type == "editor" {
			h.Broadcast <- BroadcastMessage{
				RoomID:  c.RoomID,
				Message: broadcastMsg,
				Sender:  c.UserID,
			}
		} else if msg.Type == "webrtc-offer" || msg.Type == "webrtc-answer" || msg.Type == "webrtc-candidate" {
			h.ChatCast <- ChatBroadcast{
				RoomID:  c.RoomID,
				Message: broadcastMsg,
				Sender:  c.UserID,
			}
		} else {
			log.Printf("Invalid message type: %s", msg.Type)
		}
	}
}

func (h *Hub) WriteMessages(c *Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.Unregister <- c
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed: %v", err)
				return
			}
		}
	}
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[int64]map[int64]*Connection),
		Register:   make(chan *Connection),
		Unregister: make(chan *Connection),
		Broadcast:  make(chan BroadcastMessage),
		ChatCast:   make(chan ChatBroadcast),
	}
}

func (h *Hub) Run() {
	for {

		select {
		case conn := <-h.Register:
			// New connection to room
			h.mutex.Lock()
			if h.Rooms[conn.RoomID] == nil {
				h.Rooms[conn.RoomID] = make(map[int64]*Connection)
			}
			h.Rooms[conn.RoomID][conn.UserID] = conn
			h.mutex.Unlock()
			log.Printf("User %d joined room %d", conn.UserID, conn.RoomID)

		case conn := <-h.Unregister:
			// Remove connection from room
			h.mutex.Lock()
			if room, exists := h.Rooms[conn.RoomID]; exists {
				if _, exists := room[conn.UserID]; exists {
					delete(room, conn.UserID)
					close(conn.Send)
					log.Printf("User %d left room %d", conn.UserID, conn.RoomID)
				}
			}
			h.mutex.Unlock()

		case msg := <-h.Broadcast:
			h.mutex.Lock()
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
			h.mutex.Unlock()
		case msg := <-h.ChatCast:
			h.mutex.Lock()
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
			h.mutex.Unlock()
		}
	}
}
