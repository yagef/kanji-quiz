package ws

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a Client with a buffered Send channel but no real WS conn.
func newTestClient(sessionID, participantID uuid.UUID) *Client {
	return &Client{
		Conn:          nil,
		Send:          make(chan []byte, 16),
		SessionID:     sessionID,
		ParticipantID: participantID,
	}
}

// ---------------------------------------------------------------------------
// NewSessionHub
// ---------------------------------------------------------------------------

func TestNewSessionHub(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	require.NotNil(t, h)
	assert.Equal(t, sID, h.sessionID)
	assert.NotNil(t, h.clients)
	assert.Len(t, h.clients, 0)
}

// ---------------------------------------------------------------------------
// AddClient
// ---------------------------------------------------------------------------

func TestSessionHub_AddClient(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)

	pID := uuid.New()
	c := newTestClient(sID, pID)
	h.AddClient(c)

	h.mu.RLock()
	got, ok := h.clients[pID]
	h.mu.RUnlock()
	assert.True(t, ok, "client should be registered")
	assert.Equal(t, c, got)
}

func TestSessionHub_AddClient_OverwritesPrevious(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	pID := uuid.New()

	c1 := newTestClient(sID, pID)
	c2 := newTestClient(sID, pID)
	h.AddClient(c1)
	h.AddClient(c2)

	h.mu.RLock()
	got := h.clients[pID]
	h.mu.RUnlock()
	// Second registration wins
	assert.Equal(t, c2, got)
}

func TestSessionHub_AddMultipleClients(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, pid := range ids {
		h.AddClient(newTestClient(sID, pid))
	}

	h.mu.RLock()
	assert.Len(t, h.clients, 3)
	h.mu.RUnlock()
}

// ---------------------------------------------------------------------------
// RemoveClient
// ---------------------------------------------------------------------------

func TestSessionHub_RemoveClient(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	pID := uuid.New()
	c := newTestClient(sID, pID)
	h.AddClient(c)

	h.RemoveClient(pID)

	h.mu.RLock()
	_, ok := h.clients[pID]
	h.mu.RUnlock()
	assert.False(t, ok, "client should have been removed")

	// Send channel must be closed
	_, open := <-c.Send
	assert.False(t, open, "Send channel should be closed after removal")
}

func TestSessionHub_RemoveClient_NonExistent(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	// Should not panic when removing a client that was never added
	assert.NotPanics(t, func() {
		h.RemoveClient(uuid.New())
	})
}

// ---------------------------------------------------------------------------
// Broadcast
// ---------------------------------------------------------------------------

func TestSessionHub_Broadcast_DeliveredToAll(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)

	var clients []*Client
	for i := 0; i < 3; i++ {
		c := newTestClient(sID, uuid.New())
		h.AddClient(c)
		clients = append(clients, c)
	}

	msg := []byte(`{"type":"state_sync"}`)
	h.Broadcast(msg)

	for _, c := range clients {
		select {
		case got := <-c.Send:
			assert.Equal(t, msg, got)
		default:
			t.Errorf("client %s did not receive broadcast", c.ParticipantID)
		}
	}
}

func TestSessionHub_Broadcast_NoClients(t *testing.T) {
	h := NewSessionHub(uuid.New())
	// Broadcasting with no clients must not panic
	assert.NotPanics(t, func() {
		h.Broadcast([]byte("hello"))
	})
}

func TestSessionHub_Broadcast_DropsSlowClient(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	pID := uuid.New()

	// Create a client with a zero-buffer channel so it is immediately "slow"
	slow := &Client{
		Send:          make(chan []byte, 0),
		SessionID:     sID,
		ParticipantID: pID,
	}
	h.AddClient(slow)

	// Must not block or panic; the slow client is simply dropped
	assert.NotPanics(t, func() {
		h.Broadcast([]byte("msg"))
	})
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestSessionHub_ConcurrentAddRemove(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for i := 0; i < n; i++ {
		pID := uuid.New()
		go func(id uuid.UUID) {
			defer wg.Done()
			h.AddClient(newTestClient(sID, id))
		}(pID)
		go func(id uuid.UUID) {
			defer wg.Done()
			h.RemoveClient(id)
		}(pID)
	}
	wg.Wait()
	// No data race — the race detector enforces this
}

func TestSessionHub_ConcurrentBroadcast(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	for i := 0; i < 5; i++ {
		c := newTestClient(sID, uuid.New())
		h.AddClient(c)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Broadcast([]byte("ping"))
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// SendToParticipant (lives in manager.go but operates on SessionHub)
// ---------------------------------------------------------------------------

func TestSessionHub_SendToParticipant_Delivers(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	pID := uuid.New()
	c := newTestClient(sID, pID)
	h.AddClient(c)

	msg := []byte("hello participant")
	h.SendToParticipant(pID, msg)

	select {
	case got := <-c.Send:
		assert.Equal(t, msg, got)
	default:
		t.Fatal("message was not delivered")
	}
}

func TestSessionHub_SendToParticipant_UnknownParticipant(t *testing.T) {
	h := NewSessionHub(uuid.New())
	// Unknown participant — must not panic
	assert.NotPanics(t, func() {
		h.SendToParticipant(uuid.New(), []byte("msg"))
	})
}

func TestSessionHub_SendToParticipant_DropWhenFull(t *testing.T) {
	sID := uuid.New()
	h := NewSessionHub(sID)
	pID := uuid.New()
	// zero-buffer channel
	c := &Client{Send: make(chan []byte, 0), SessionID: sID, ParticipantID: pID}
	h.AddClient(c)

	// Should not block or panic
	assert.NotPanics(t, func() {
		h.SendToParticipant(pID, []byte("msg"))
	})
}
