package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// MessageType constants
// ---------------------------------------------------------------------------

func TestMessageTypeConstants(t *testing.T) {
	assert.Equal(t, MessageType("state_sync"), MsgStateSync)
	assert.Equal(t, MessageType("answer"), MsgAnswer)
	assert.Equal(t, MessageType("error"), MsgError)
	assert.Equal(t, MessageType("heartbeat"), MsgHeartbeat)
}

// ---------------------------------------------------------------------------
// StatePhase constants
// ---------------------------------------------------------------------------

func TestStatePhaseConstants(t *testing.T) {
	phases := []StatePhase{PhaseWaiting, PhaseCountdown, PhaseAnswering, PhasePaused, PhaseFinished}
	expected := []string{"waiting", "countdown", "answering", "paused", "finished"}
	for i, p := range phases {
		assert.Equal(t, StatePhase(expected[i]), p)
	}
	// All phases must be distinct
	seen := map[StatePhase]bool{}
	for _, p := range phases {
		assert.False(t, seen[p], "duplicate phase value: %s", p)
		seen[p] = true
	}
}

// ---------------------------------------------------------------------------
// Envelope marshal / unmarshal round-trip
// ---------------------------------------------------------------------------

func TestEnvelopeRoundTrip(t *testing.T) {
	inner := AnswerPayload{
		QuestionID: uuid.New().String(),
		AnswerID:   uuid.New().String(),
	}
	rawInner, err := json.Marshal(inner)
	require.NoError(t, err)

	env := Envelope{
		Type:    MsgAnswer,
		Payload: rawInner,
	}
	data, err := json.Marshal(env)
	require.NoError(t, err)

	var decoded Envelope
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, MsgAnswer, decoded.Type)

	var got AnswerPayload
	require.NoError(t, json.Unmarshal(decoded.Payload, &got))
	assert.Equal(t, inner.QuestionID, got.QuestionID)
	assert.Equal(t, inner.AnswerID, got.AnswerID)
}

func TestEnvelopeUnknownType(t *testing.T) {
	raw := `{"type":"unknown_type","payload":{"foo":"bar"}}`
	var env Envelope
	require.NoError(t, json.Unmarshal([]byte(raw), &env))
	assert.Equal(t, MessageType("unknown_type"), env.Type)
	// Payload must be preserved verbatim
	assert.Contains(t, string(env.Payload), "foo")
}

// ---------------------------------------------------------------------------
// StateSyncPayload JSON serialization
// ---------------------------------------------------------------------------

func TestStateSyncPayload_JSONFieldNames(t *testing.T) {
	qID := uuid.New().String()
	p := StateSyncPayload{
		Phase:              PhaseAnswering,
		QuestionID:         qID,
		QuestionText:       "読み方 日",
		Answers:            []AnswerVO{{ID: uuid.New().String(), Text: "にち"}},
		RemainingSeconds:   8,
		TotalSeconds:       15,
		QuestionIndex:      2,
		TotalQuestions:     10,
		Score:              750,
		HasAnsweredCurrent: true,
	}
	data, err := json.Marshal(p)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "answering", m["phase"])
	assert.Equal(t, qID, m["question_id"])
	assert.Equal(t, "読み方 日", m["question_text"])
	assert.Equal(t, float64(8), m["remaining_seconds"])
	assert.Equal(t, float64(15), m["total_seconds"])
	assert.Equal(t, float64(2), m["question_index"])
	assert.Equal(t, float64(10), m["total_questions"])
	assert.Equal(t, float64(750), m["score"])
	assert.Equal(t, true, m["has_answered_current"])
}

func TestStateSyncPayload_OmitsEmptyOptionals(t *testing.T) {
	// When waiting, question_id / question_text / answers should be absent
	p := StateSyncPayload{
		Phase:          PhaseWaiting,
		TotalQuestions: 5,
	}
	data, err := json.Marshal(p)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	_, hasQID := m["question_id"]
	_, hasQText := m["question_text"]
	_, hasAnswers := m["answers"]
	assert.False(t, hasQID, "question_id should be omitted when empty")
	assert.False(t, hasQText, "question_text should be omitted when empty")
	assert.False(t, hasAnswers, "answers should be omitted when nil")
}

// ---------------------------------------------------------------------------
// AnswerPayload
// ---------------------------------------------------------------------------

func TestAnswerPayload_RoundTrip(t *testing.T) {
	qID := uuid.New().String()
	aID := uuid.New().String()
	orig := AnswerPayload{QuestionID: qID, AnswerID: aID}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var got AnswerPayload
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, orig, got)
}

// ---------------------------------------------------------------------------
// ErrorPayload
// ---------------------------------------------------------------------------

func TestErrorPayload_RoundTrip(t *testing.T) {
	orig := ErrorPayload{Message: "something went wrong"}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var got ErrorPayload
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, orig, got)
}

// ---------------------------------------------------------------------------
// QuestionRound zero value
// ---------------------------------------------------------------------------

func TestQuestionRound_ZeroValue(t *testing.T) {
	var r QuestionRound
	assert.Equal(t, uuid.UUID{}, r.QuestionID)
	assert.Nil(t, r.AnswerIDs)
	assert.Zero(t, r.Index)
	assert.Zero(t, r.TimeLimit)
	assert.True(t, r.Deadline.IsZero())
}

// ---------------------------------------------------------------------------
// SessionState zero value
// ---------------------------------------------------------------------------

func TestSessionState_ZeroValue(t *testing.T) {
	var s SessionState
	assert.Equal(t, StatePhase(""), s.Phase)
	assert.Equal(t, 0, s.CurrentIndex)
	assert.Nil(t, s.Rounds)
	assert.Zero(t, s.CountdownDuration)
	assert.Zero(t, s.AnswerDuration)
}

// ---------------------------------------------------------------------------
// AnswerVO
// ---------------------------------------------------------------------------

func TestAnswerVO_JSONRoundTrip(t *testing.T) {
	orig := AnswerVO{ID: uuid.New().String(), Text: "東京"}
	data, err := json.Marshal(orig)
	require.NoError(t, err)
	var got AnswerVO
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, orig, got)
}

// ---------------------------------------------------------------------------
// Envelope with StateSyncPayload (full integration)
// ---------------------------------------------------------------------------

func TestEnvelope_StateSyncPayload(t *testing.T) {
	syncPayload := StateSyncPayload{
		Phase:            PhaseCountdown,
		RemainingSeconds: 3,
		TotalSeconds:     5,
		QuestionIndex:    1,
		TotalQuestions:   3,
	}
	rawPayload, err := json.Marshal(syncPayload)
	require.NoError(t, err)

	env := Envelope{Type: MsgStateSync, Payload: rawPayload}
	data, err := json.Marshal(env)
	require.NoError(t, err)

	var decoded Envelope
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, MsgStateSync, decoded.Type)

	var got StateSyncPayload
	require.NoError(t, json.Unmarshal(decoded.Payload, &got))
	assert.Equal(t, PhaseCountdown, got.Phase)
	assert.Equal(t, 3, got.RemainingSeconds)
	assert.Equal(t, 5, got.TotalSeconds)
}

// ---------------------------------------------------------------------------
// QuestionRound fields survive assignment
// ---------------------------------------------------------------------------

func TestQuestionRound_FieldAssignment(t *testing.T) {
	qID := uuid.New()
	aID1 := uuid.New()
	aID2 := uuid.New()
	deadline := time.Now().Add(10 * time.Second)

	r := QuestionRound{
		QuestionID: qID,
		AnswerIDs:  []uuid.UUID{aID1, aID2},
		TimeLimit:  15 * time.Second,
		Deadline:   deadline,
		Index:      3,
	}

	assert.Equal(t, qID, r.QuestionID)
	assert.Equal(t, []uuid.UUID{aID1, aID2}, r.AnswerIDs)
	assert.Equal(t, 15*time.Second, r.TimeLimit)
	assert.Equal(t, deadline, r.Deadline)
	assert.Equal(t, 3, r.Index)
}
