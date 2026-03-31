package repository

import (
	"context"
	"kanji-quiz/server/model"

	"github.com/google/uuid"
)

// QuizRepoIface is the subset of QuizRepo methods consumed by the ws engine.
// It allows test code to inject a mock without a real database.
type QuizRepoIface interface {
	ListQuestions(ctx context.Context, quizID uuid.UUID) ([]model.Question, error)
	PickRandomAnswersForQuestion(ctx context.Context, questionID uuid.UUID, count int) ([]uuid.UUID, error)
	GetQuestion(ctx context.Context, id uuid.UUID) (model.Question, error)
	GetAnswersByIDs(ctx context.Context, answerIDs []uuid.UUID) ([]model.Answer, error)
	ListParticipants(ctx context.Context, sessionID uuid.UUID) ([]model.Participant, error)
	GetParticipantScore(ctx context.Context, participantID uuid.UUID) (int, error)
	HasParticipantAnswered(ctx context.Context, participantID, questionID uuid.UUID) (bool, error)
	InsertTimeoutSubmissions(ctx context.Context, sessionID, questionID uuid.UUID, timeLimitMs int) error
	EndSession(ctx context.Context, sessionID uuid.UUID) error
	CountParticipants(ctx context.Context, sessionID uuid.UUID) (int, error)
	CountSubmissionsForQuestion(ctx context.Context, sessionID, questionID uuid.UUID) (int, error)
	InsertSubmissionAndUpdateScore(ctx context.Context, participantID, questionID, answerID uuid.UUID, isCorrect bool, timeTakenMs int, timeLimit int) error
	IsAnswerCorrect(ctx context.Context, questionID, answerID uuid.UUID) (bool, error)
}

// Compile-time check: *QuizRepo must satisfy the interface.
var _ QuizRepoIface = (*QuizRepo)(nil)
