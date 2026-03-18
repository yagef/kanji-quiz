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

// GetUserHistory returns all sessions the user participated in, newest first.
func (r *QuizRepo) GetUserHistory(ctx context.Context, userName string) ([]model.HistoryEntry, error) {
	rows, err := r.db.Query(ctx, `
        SELECT
            p.id,
            qs.id,
            q.title,
            qs.started_at,
            qs.ended_at,
            p.score
        FROM participants p
        JOIN quiz_sessions qs ON p.session_id = qs.id
        JOIN quizzes       q  ON qs.quiz_id   = q.id
        JOIN users         u  ON p.user_id     = u.id
        WHERE lower(u.name) = lower($1)
        ORDER BY qs.started_at DESC
    `, userName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.HistoryEntry
	for rows.Next() {
		var e model.HistoryEntry
		if err := rows.Scan(&e.ParticipantID, &e.SessionID, &e.QuizTitle,
			&e.StartedAt, &e.EndedAt, &e.Score); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// GetParticipantSubmissions returns every answered question for a participant.
func (r *QuizRepo) GetParticipantSubmissions(ctx context.Context, participantID uuid.UUID) ([]model.SubmissionDetail, error) {
	rows, err := r.db.Query(ctx, `
        SELECT
            qu.kanji,
            at.title,
            COALESCE(given_ans.text, '— (no answer)'),
            correct_ans.text,
            s.is_correct
        FROM submissions s
        JOIN questions     qu          ON s.question_id          = qu.id
        JOIN answer_types  at          ON qu.type_id              = at.id
        LEFT JOIN answers  given_ans   ON s.answer_id             = given_ans.id
        JOIN answers       correct_ans ON qu.correct_answer_id    = correct_ans.id
        WHERE s.participant_id = $1
        ORDER BY qu.kanji
    `, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SubmissionDetail
	for rows.Next() {
		var d model.SubmissionDetail
		if err := rows.Scan(&d.Kanji, &d.QuestionType,
			&d.AnswerGiven, &d.CorrectAnswer, &d.IsCorrect); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}
