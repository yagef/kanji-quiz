package repository

import (
	"context"
	"kanji-quiz/server/model"

	"github.com/google/uuid"
)

// internal/repository/user_repo.go (or in the same repo if you prefer)

func (r *QuizRepo) GetOrCreateUserByName(ctx context.Context, name string) (model.User, error) {
	var u model.User

	// Normalize name same way as your unique index (lower(name))
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (name)
		VALUES ($1)
		ON CONFLICT (lower(name)) DO UPDATE
		SET name = EXCLUDED.name
		RETURNING id, name
	`, name).
		Scan(&u.ID, &u.Name)

	return u, err
}

func (r *QuizRepo) GetParticipantByUserAndSession(ctx context.Context, userID, sessionID uuid.UUID) (model.Participant, error) {
	var p model.Participant

	err := r.db.QueryRow(ctx, `
		SELECT p.id, p.user_id, u.name, p.score, p.session_id
		FROM participants p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = $1 AND p.session_id = $2
	`, userID, sessionID).
		Scan(&p.ID, &p.UserID, &p.Name, &p.Score, &p.SessionID)

	return p, err
}

func (r *QuizRepo) CreateParticipant(ctx context.Context, userID, sessionID uuid.UUID) (model.Participant, error) {
	var p model.Participant

	err := r.db.QueryRow(ctx, `
		INSERT INTO participants (user_id, session_id)
		VALUES ($1, $2)
		RETURNING id, user_id
	`, userID, sessionID).Scan(&p.ID, &p.UserID)

	// You likely want the name and score too:
	err = r.db.QueryRow(ctx, `
		SELECT p.id, p.user_id, u.name, p.score, p.session_id
		FROM participants p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1
	`, p.ID).Scan(&p.ID, &p.UserID, &p.Name, &p.Score, &p.SessionID)

	return p, err
}

func (r *QuizRepo) GetParticipant(ctx context.Context, participantID uuid.UUID) (model.Participant, error) {
	var p model.Participant

	err := r.db.QueryRow(ctx, `
		SELECT p.id, p.user_id, u.name, p.score, p.session_id
		FROM participants p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1
	`, participantID).
		Scan(&p.ID, &p.UserID, &p.Name, &p.Score, &p.SessionID)

	return p, err
}
