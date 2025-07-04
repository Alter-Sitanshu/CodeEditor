package sockets

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// VoiceChat represents a voice chat session in a room
type VoiceChat struct {
	RoomID       int64                       `json:"room_id"`
	Participants map[int64]*VoiceParticipant `json:"participants"`
	Active       bool                        `json:"active"`
	StartedAt    time.Time                   `json:"started_at"`
	mutex        sync.RWMutex
}

// VoiceParticipant represents a user in voice chat
type VoiceParticipant struct {
	UserID     int64     `json:"user_id"`
	Muted      bool      `json:"muted"`
	Deafened   bool      `json:"deafened"`
	JoinedAt   time.Time `json:"joined_at"`
	Speaking   bool      `json:"speaking"`
	AudioLevel float64   `json:"audio_level"`
}

// VoiceChatManager manages voice chats across rooms
type VoiceChatManager struct {
	VoiceChats map[int64]*VoiceChat `json:"voice_chats"`
	mutex      sync.RWMutex
}

// Voice chat message types
type VoiceMessage struct {
	Type      string `json:"type"`
	RoomID    int64  `json:"room_id"`
	UserID    int64  `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data"`
}

// WebRTC signaling data structures
type WebRTCOffer struct {
	SDP        string `json:"sdp"`
	TargetUser int64  `json:"target_user"`
}

type WebRTCAnswer struct {
	SDP        string `json:"sdp"`
	TargetUser int64  `json:"target_user"`
}

type WebRTCCandidate struct {
	Candidate  string `json:"candidate"`
	TargetUser int64  `json:"target_user"`
}

// Voice state changes
type VoiceStateUpdate struct {
	UserID     int64   `json:"user_id"`
	Muted      bool    `json:"muted"`
	Deafened   bool    `json:"deafened"`
	Speaking   bool    `json:"speaking"`
	AudioLevel float64 `json:"audio_level"`
}

// Voice chat events
type VoiceJoinRequest struct {
	UserID int64 `json:"user_id"`
}

type VoiceLeaveRequest struct {
	UserID int64 `json:"user_id"`
}

// NewVoiceChatManager creates a new voice chat manager
func NewVoiceChatManager() *VoiceChatManager {
	return &VoiceChatManager{
		VoiceChats: make(map[int64]*VoiceChat),
	}
}

// JoinVoiceChat adds a user to voice chat in a room
func (vcm *VoiceChatManager) JoinVoiceChat(roomID, userID int64) *VoiceParticipant {
	vcm.mutex.Lock()
	defer vcm.mutex.Unlock()

	// Create voice chat if it doesn't exist
	if vcm.VoiceChats[roomID] == nil {
		vcm.VoiceChats[roomID] = &VoiceChat{
			RoomID:       roomID,
			Participants: make(map[int64]*VoiceParticipant),
			Active:       true,
			StartedAt:    time.Now(),
		}
	}

	voiceChat := vcm.VoiceChats[roomID]
	voiceChat.mutex.Lock()
	defer voiceChat.mutex.Unlock()

	// Add participant
	participant := &VoiceParticipant{
		UserID:     userID,
		Muted:      false,
		Deafened:   false,
		JoinedAt:   time.Now(),
		Speaking:   false,
		AudioLevel: 0.0,
	}

	voiceChat.Participants[userID] = participant
	log.Printf("User %d joined voice chat in room %d", userID, roomID)

	return participant
}

// LeaveVoiceChat removes a user from voice chat
func (vcm *VoiceChatManager) LeaveVoiceChat(roomID, userID int64) {
	vcm.mutex.Lock()
	defer vcm.mutex.Unlock()

	if voiceChat, exists := vcm.VoiceChats[roomID]; exists {
		voiceChat.mutex.Lock()
		defer voiceChat.mutex.Unlock()

		delete(voiceChat.Participants, userID)
		log.Printf("User %d left voice chat in room %d", userID, roomID)

		// Clean up empty voice chats
		if len(voiceChat.Participants) == 0 {
			delete(vcm.VoiceChats, roomID)
			log.Printf("Voice chat ended in room %d", roomID)
		}
	}
}

// UpdateVoiceState updates a participant's voice state
func (vcm *VoiceChatManager) UpdateVoiceState(roomID, userID int64, update VoiceStateUpdate) {
	vcm.mutex.RLock()
	defer vcm.mutex.RUnlock()

	if voiceChat, exists := vcm.VoiceChats[roomID]; exists {
		voiceChat.mutex.Lock()
		defer voiceChat.mutex.Unlock()

		if participant, exists := voiceChat.Participants[userID]; exists {
			participant.Muted = update.Muted
			participant.Deafened = update.Deafened
			participant.Speaking = update.Speaking
			participant.AudioLevel = update.AudioLevel
		}
	}
}

// GetVoiceChat returns the voice chat for a room
func (vcm *VoiceChatManager) GetVoiceChat(roomID int64) *VoiceChat {
	vcm.mutex.RLock()
	defer vcm.mutex.RUnlock()

	if voiceChat, exists := vcm.VoiceChats[roomID]; exists {
		return voiceChat
	}
	return nil
}

// GetParticipants returns all participants in a room's voice chat
func (vcm *VoiceChatManager) GetParticipants(roomID int64) []*VoiceParticipant {
	vcm.mutex.RLock()
	defer vcm.mutex.RUnlock()

	if voiceChat, exists := vcm.VoiceChats[roomID]; exists {
		voiceChat.mutex.RLock()
		defer voiceChat.mutex.RUnlock()

		participants := make([]*VoiceParticipant, 0, len(voiceChat.Participants))
		for _, participant := range voiceChat.Participants {
			participants = append(participants, participant)
		}
		return participants
	}
	return nil
}

// HandleVoiceMessage processes voice-related WebSocket messages
func (h *Hub) HandleVoiceMessage(c *Connection, msg WSMessage, vcm *VoiceChatManager) {
	switch msg.Type {
	case "voice-join":
		var joinReq VoiceJoinRequest
		if data, err := json.Marshal(msg.Data); err == nil {
			if err := json.Unmarshal(data, &joinReq); err == nil {

				participant := vcm.JoinVoiceChat(c.RoomID, c.UserID)

				// Broadcast to all room members that user joined voice
				response := WSMessage{
					Type:      "voice-user-joined",
					RoomID:    c.RoomID,
					UserID:    c.UserID,
					Timestamp: time.Now().Unix(),
					Data:      participant,
				}

				broadcastMsg, _ := json.Marshal(response)
				h.ChatCast <- ChatBroadcast{
					RoomID:  c.RoomID,
					Message: broadcastMsg,
					Sender:  c.UserID,
				}
			}
		}

	case "voice-leave":
		vcm.LeaveVoiceChat(c.RoomID, c.UserID)

		// Broadcast to all room members that user left voice
		response := WSMessage{
			Type:      "voice-user-left",
			RoomID:    c.RoomID,
			UserID:    c.UserID,
			Timestamp: time.Now().Unix(),
			Data:      map[string]int64{"user_id": c.UserID},
		}

		broadcastMsg, _ := json.Marshal(response)
		h.ChatCast <- ChatBroadcast{
			RoomID:  c.RoomID,
			Message: broadcastMsg,
			Sender:  c.UserID,
		}

	case "voice-state-update":
		var stateUpdate VoiceStateUpdate
		if data, err := json.Marshal(msg.Data); err == nil {
			if err := json.Unmarshal(data, &stateUpdate); err == nil {
				stateUpdate.UserID = c.UserID
				vcm.UpdateVoiceState(c.RoomID, c.UserID, stateUpdate)

				// Broadcast state update to all room members
				response := WSMessage{
					Type:      "voice-state-updated",
					RoomID:    c.RoomID,
					UserID:    c.UserID,
					Timestamp: time.Now().Unix(),
					Data:      stateUpdate,
				}

				broadcastMsg, _ := json.Marshal(response)
				h.ChatCast <- ChatBroadcast{
					RoomID:  c.RoomID,
					Message: broadcastMsg,
					Sender:  c.UserID,
				}
			}
		}

	case "voice-participants":
		// Send current participants to requesting user
		participants := vcm.GetParticipants(c.RoomID)
		response := WSMessage{
			Type:      "voice-participants-list",
			RoomID:    c.RoomID,
			UserID:    c.UserID,
			Timestamp: time.Now().Unix(),
			Data:      participants,
		}

		responseMsg, _ := json.Marshal(response)
		select {
		case c.Send <- responseMsg:
		default:
			// Connection is dead
			vcm.LeaveVoiceChat(c.RoomID, c.UserID)
			close(c.Send)
		}
	}
}
