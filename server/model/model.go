package model

import "github.com/google/uuid"

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
	Kanji           string
	CorrectAnswerID *uuid.UUID // nullable — set after answers are added
}

type Answer struct {
	ID         uuid.UUID
	QuestionID uuid.UUID
	Text       string
}

type QuizSession struct {
	ID     uuid.UUID
	QuizID uuid.UUID
	Token  uuid.UUID
}
