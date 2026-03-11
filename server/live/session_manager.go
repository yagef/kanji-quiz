package live

import (
	"kanji-quiz/server/model"
	"sync"

	"github.com/google/uuid"
)

type Manager struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]*model.SessionState
}

func NewManager() *Manager {
	return &Manager{
		byID: make(map[uuid.UUID]*model.SessionState),
	}
}

func (m *Manager) Get(sessionID uuid.UUID) (*model.SessionState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.byID[sessionID]
	return s, ok
}

func (m *Manager) Set(state *model.SessionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[state.SessionID] = state
}

func (m *Manager) Delete(sessionID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, sessionID)
}
