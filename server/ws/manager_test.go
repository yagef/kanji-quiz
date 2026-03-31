package ws

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewManager
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	m := NewManager()
	require.NotNil(t, m)
	assert.NotNil(t, m.hubs)
	assert.Len(t, m.hubs, 0)
}

// ---------------------------------------------------------------------------
// GetOrCreate
// ---------------------------------------------------------------------------

func TestManager_GetOrCreate_CreatesNewHub(t *testing.T) {
	m := NewManager()
	sID := uuid.New()

	hub := m.GetOrCreate(sID)
	require.NotNil(t, hub)
	assert.Equal(t, sID, hub.sessionID)
}

func TestManager_GetOrCreate_ReturnsSameHub(t *testing.T) {
	m := NewManager()
	sID := uuid.New()

	h1 := m.GetOrCreate(sID)
	h2 := m.GetOrCreate(sID)
	assert.Same(t, h1, h2, "should return the same hub pointer for the same session")
}

func TestManager_GetOrCreate_DifferentSessionsGetDifferentHubs(t *testing.T) {
	m := NewManager()
	h1 := m.GetOrCreate(uuid.New())
	h2 := m.GetOrCreate(uuid.New())
	assert.NotSame(t, h1, h2)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestManager_Delete(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	m.GetOrCreate(sID)

	m.Delete(sID)

	m.mu.RLock()
	_, exists := m.hubs[sID]
	m.mu.RUnlock()
	assert.False(t, exists, "hub should have been deleted")
}

func TestManager_Delete_NonExistent(t *testing.T) {
	m := NewManager()
	assert.NotPanics(t, func() {
		m.Delete(uuid.New())
	})
}

func TestManager_Delete_ThenGetOrCreate_BuildsFresh(t *testing.T) {
	m := NewManager()
	sID := uuid.New()

	h1 := m.GetOrCreate(sID)
	m.Delete(sID)
	h2 := m.GetOrCreate(sID)

	assert.NotSame(t, h1, h2, "after delete a new hub should be created")
}

// ---------------------------------------------------------------------------
// RemoveIfEmpty
// ---------------------------------------------------------------------------

func TestManager_RemoveIfEmpty_RemovesEmptyHub(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	m.GetOrCreate(sID)

	m.RemoveIfEmpty(sID)

	m.mu.RLock()
	_, exists := m.hubs[sID]
	m.mu.RUnlock()
	assert.False(t, exists)
}

func TestManager_RemoveIfEmpty_KeepsNonEmptyHub(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	h := m.GetOrCreate(sID)

	// Add a client so the hub is not empty
	h.AddClient(newTestClient(sID, uuid.New()))

	m.RemoveIfEmpty(sID)

	m.mu.RLock()
	_, exists := m.hubs[sID]
	m.mu.RUnlock()
	assert.True(t, exists, "non-empty hub must not be removed")
}

func TestManager_RemoveIfEmpty_NonExistent(t *testing.T) {
	m := NewManager()
	assert.NotPanics(t, func() {
		m.RemoveIfEmpty(uuid.New())
	})
}

// ---------------------------------------------------------------------------
// IsParticipantConnected
// ---------------------------------------------------------------------------

func TestManager_IsParticipantConnected_True(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	pID := uuid.New()

	h := m.GetOrCreate(sID)
	h.AddClient(newTestClient(sID, pID))

	assert.True(t, m.IsParticipantConnected(sID, pID))
}

func TestManager_IsParticipantConnected_FalseWhenNoHub(t *testing.T) {
	m := NewManager()
	assert.False(t, m.IsParticipantConnected(uuid.New(), uuid.New()))
}

func TestManager_IsParticipantConnected_FalseWhenClientAbsent(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	m.GetOrCreate(sID) // hub exists but no clients

	assert.False(t, m.IsParticipantConnected(sID, uuid.New()))
}

func TestManager_IsParticipantConnected_FalseAfterRemoval(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	pID := uuid.New()

	h := m.GetOrCreate(sID)
	h.AddClient(newTestClient(sID, pID))
	assert.True(t, m.IsParticipantConnected(sID, pID))

	h.RemoveClient(pID)
	assert.False(t, m.IsParticipantConnected(sID, pID))
}

// ---------------------------------------------------------------------------
// ConnectedCount
// ---------------------------------------------------------------------------

func TestManager_ConnectedCount_Zero_NoHub(t *testing.T) {
	m := NewManager()
	assert.Equal(t, 0, m.ConnectedCount(uuid.New()))
}

func TestManager_ConnectedCount_Zero_EmptyHub(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	m.GetOrCreate(sID)
	assert.Equal(t, 0, m.ConnectedCount(sID))
}

func TestManager_ConnectedCount_MatchesClients(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	h := m.GetOrCreate(sID)

	const n = 5
	for i := 0; i < n; i++ {
		h.AddClient(newTestClient(sID, uuid.New()))
	}
	assert.Equal(t, n, m.ConnectedCount(sID))
}

func TestManager_ConnectedCount_DecreasesAfterRemoval(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	h := m.GetOrCreate(sID)

	pID := uuid.New()
	h.AddClient(newTestClient(sID, pID))
	h.AddClient(newTestClient(sID, uuid.New()))
	assert.Equal(t, 2, m.ConnectedCount(sID))

	h.RemoveClient(pID)
	assert.Equal(t, 1, m.ConnectedCount(sID))
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestManager_ConcurrentGetOrCreate(t *testing.T) {
	m := NewManager()
	sID := uuid.New()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h := m.GetOrCreate(sID)
			assert.NotNil(t, h)
		}()
	}
	wg.Wait()
}

func TestManager_ConcurrentDeleteAndGetOrCreate(t *testing.T) {
	m := NewManager()
	sID := uuid.New()
	m.GetOrCreate(sID)

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.Delete(sID)
		}()
		go func() {
			defer wg.Done()
			m.GetOrCreate(sID)
		}()
	}
	wg.Wait()
}
