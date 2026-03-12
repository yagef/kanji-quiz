package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MsgStateSync MessageType = "state_sync" // server → client: full state for current question
	MsgAnswer    MessageType = "answer"     // client → server: player answer
	MsgError     MessageType = "error"      // server → client: error string
	MsgHeartbeat MessageType = "heartbeat"  // optional, ping/pong
)

type StatePhase string

const (
	PhaseWaiting   StatePhase = "waiting"
	PhaseCountdown StatePhase = "countdown"
	PhaseAnswering StatePhase = "answering"
	PhasePaused    StatePhase = "paused"
	PhaseFinished  StatePhase = "finished"
)

type StateSyncPayload struct {
	Phase              StatePhase `json:"phase"`
	QuestionID         string     `json:"question_id,omitempty"`
	QuestionText       string     `json:"question_text,omitempty"`
	Answers            []AnswerVO `json:"answers,omitempty"`
	RemainingSeconds   int        `json:"remaining_seconds,omitempty"`
	TotalSeconds       int        `json:"total_seconds,omitempty"`
	QuestionIndex      int        `json:"question_index"`
	TotalQuestions     int        `json:"total_questions"`
	Score              int        `json:"score"`
	HasAnsweredCurrent bool       `json:"has_answered_current"`
}

type QuestionRound struct {
	QuestionID uuid.UUID
	AnswerIDs  []uuid.UUID

	TimeLimit time.Duration
	Deadline  time.Time
	Index     int
}

type SessionState struct {
	SessionID uuid.UUID
	QuizID    uuid.UUID

	Phase        StatePhase
	Rounds       []QuestionRound
	CurrentIndex int

	CountdownDuration time.Duration // e.g. 5s
	AnswerDuration    time.Duration // per question

	mu sync.RWMutex
}

type AnswerVO struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type AnswerPayload struct {
	QuestionID string `json:"question_id"`
	AnswerID   string `json:"answer_id"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type Envelope struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
