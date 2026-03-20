package ws

import (
	"sync"

	"github.com/google/uuid"
)

type Manager struct {
	mu   sync.RWMutex
	hubs map[uuid.UUID]*SessionHub
}

func NewManager() *Manager {
	return &Manager{hubs: make(map[uuid.UUID]*SessionHub)}
}

func (m *Manager) GetOrCreate(sessionID uuid.UUID) *SessionHub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.hubs[sessionID]; ok {
		return h
	}
	h := NewSessionHub(sessionID)
	m.hubs[sessionID] = h
	return h
}

func (m *Manager) RemoveIfEmpty(sessionID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.hubs[sessionID]; ok {
		h.mu.RLock()
		empty := len(h.clients) == 0
		h.mu.RUnlock()
		if empty {
			delete(m.hubs, sessionID)
		}
	}
}

func (m *Manager) Delete(sessionID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hubs[sessionID]; ok {
		delete(m.hubs, sessionID)
	}
}

// IsParticipantConnected reports whether the participant currently has an active WS connection.
func (m *Manager) IsParticipantConnected(sessionID, participantID uuid.UUID) bool {
	m.mu.RLock()
	h, ok := m.hubs[sessionID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	h.mu.RLock()
	_, connected := h.clients[participantID]
	h.mu.RUnlock()
	return connected
}

func (h *SessionHub) SendToParticipant(participantID uuid.UUID, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if c, ok := h.clients[participantID]; ok {
		select {
		case c.Send <- msg:
		default:
			// drop if buffer full
		}
	}
}
