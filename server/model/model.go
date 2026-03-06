package model

import (
	"time"

	"github.com/google/uuid"
)

type Quiz struct {
	ID    uuid.UUID
	Title string
}

type AnswerType struct {
	ID    int
	Text  string
	Title string
}

type Question struct {
	ID              uuid.UUID
	QuizID          uuid.UUID
	TypeID          int
	TypeText        string
	Kanji           string
	CorrectAnswerID *uuid.UUID // nullable — set after answers are added
}

type Answer struct {
	ID         uuid.UUID
	QuestionID uuid.UUID
	Text       string
}

type QuizSession struct {
	ID        uuid.UUID
	QuizID    uuid.UUID
	StartedAt *time.Time
	EndedAt   *time.Time
}

type Participant struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Name   string // Joined from users table
	Score  int
}
