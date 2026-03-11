package model

import (
	"sync"

	"github.com/google/uuid"
)

type Phase string

const (
	PhaseWaiting   Phase = "waiting"
	PhaseCountdown Phase = "countdown"
	PhaseAnswering Phase = "answering"
	PhasePaused    Phase = "paused"
	PhaseFinished  Phase = "finished"
)

type SessionQuestion struct {
	QuestionID uuid.UUID
	AnswerIDs  []uuid.UUID // exactly 4
}

type SessionState struct {
	SessionID        uuid.UUID
	QuizID           uuid.UUID
	Phase            Phase
	CurrentIndex     int
	Questions        []SessionQuestion // shuffled
	QuestionDuration int               // seconds
	mu               sync.RWMutex
}

type QuestionWithAnswers struct {
	Question Question
	Answers  []Answer
}
