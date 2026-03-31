package ws

import (
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn          *websocket.Conn
	Send          chan []byte
	SessionID     uuid.UUID
	ParticipantID uuid.UUID
}

type SessionHub struct {
	sessionID uuid.UUID

	mu      sync.RWMutex
	clients map[uuid.UUID]*Client // keyed by ParticipantID
}

func NewSessionHub(sessionID uuid.UUID) *SessionHub {
	return &SessionHub{
		sessionID: sessionID,
		clients:   make(map[uuid.UUID]*Client),
	}
}

func (h *SessionHub) AddClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ParticipantID] = c
}

func (h *SessionHub) RemoveClient(participantID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[participantID]; ok {
		close(c.Send)
		delete(h.clients, participantID)
	}
}

func (h *SessionHub) Broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for pid, c := range h.clients {
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow client %s", pid)
			close(c.Send)
			delete(h.clients, pid)
		}
	}
}
