package repository

import (
	"context"
	"errors"
	"fmt"
	"kanji-quiz/server/model"
	"math/rand"

	"github.com/google/uuid"
)

func (r *QuizRepo) GetSessionQuiz(ctx context.Context, sessionId uuid.UUID) (model.Quiz, error) {
	var q model.Quiz
	err := r.db.QueryRow(ctx, `SELECT q.id, q.title
		FROM quizzes AS q
		INNER JOIN quiz_sessions AS s ON s.quiz_id = q.id
		WHERE s.id = $1`, sessionId).Scan(&q.ID, &q.Title)
	return q, err
}

func (r *QuizRepo) CreateSession(ctx context.Context, quizID uuid.UUID) (model.QuizSession, error) {
	var s model.QuizSession
	err := r.db.QueryRow(ctx,
		`INSERT INTO quiz_sessions (quiz_id) VALUES ($1) RETURNING id, quiz_id, started_at, ended_at`, quizID,
	).Scan(&s.ID, &s.QuizID, &s.StartedAt, &s.EndedAt)
	return s, err
}

// EndSession Mark a session as ended
func (r *QuizRepo) EndSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE quiz_sessions SET ended_at = NOW() WHERE id = $1`, sessionID)
	return err
}

// ListSessions List sessions for a quiz
func (r *QuizRepo) ListSessions(ctx context.Context, quizID uuid.UUID) ([]model.QuizSession, error) {
	rows, err := r.db.Query(ctx, `SELECT id, quiz_id, started_at, ended_at FROM quiz_sessions WHERE quiz_id = $1 ORDER BY started_at DESC`, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.QuizSession
	for rows.Next() {
		var s model.QuizSession
		if err := rows.Scan(&s.ID, &s.QuizID, &s.StartedAt, &s.EndedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// GetSession Get single session
func (r *QuizRepo) GetSession(ctx context.Context, sessionID uuid.UUID) (model.QuizSession, error) {
	var s model.QuizSession
	err := r.db.QueryRow(ctx, `SELECT id, quiz_id, started_at, ended_at FROM quiz_sessions WHERE id = $1`, sessionID).
		Scan(&s.ID, &s.QuizID, &s.StartedAt, &s.EndedAt)
	return s, err
}

func (r *QuizRepo) DeleteSession(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `DELETE FROM submissions AS s
       USING participants AS p
       WHERE p.id = s.participant_id
       AND p.session_id = $1`, id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM participants WHERE session_id = $1`, id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM quiz_sessions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListParticipants List participants for a session
func (r *QuizRepo) ListParticipants(ctx context.Context, sessionID uuid.UUID) ([]model.Participant, error) {
	rows, err := r.db.Query(ctx, `
        SELECT p.id, p.user_id, u.name, COALESCE(SUM(s.score), 0) AS score
        FROM participants p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN submissions s ON s.participant_id = p.id
        WHERE p.session_id = $1
        GROUP BY p.id, p.user_id, u.name
        ORDER BY score DESC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Participant
	for rows.Next() {
		var p model.Participant
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Score); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// PickRandomAnswersForQuestion returns `limit` random answer IDs for a question.
func (r *QuizRepo) PickRandomAnswersForQuestion(ctx context.Context, questionID uuid.UUID, count int) ([]uuid.UUID, error) {
	// 1. Get the correct answer for this question
	var correctAnswerID uuid.UUID
	err := r.db.QueryRow(ctx, `
        SELECT q.correct_answer_id
        FROM questions AS q 
        WHERE q.id = $1
        LIMIT 1
    `, questionID).Scan(&correctAnswerID)
	if err != nil {
		return nil, fmt.Errorf("correct answer not found for question %s: %w", questionID, err)
	}

	// 2. Pick (count-1) random wrong answers
	rows, err := r.db.Query(ctx, `
        SELECT a.id
        FROM answers AS a
        INNER JOIN questions AS q ON q.id = a.question_id AND q.correct_answer_id != a.id 
        WHERE a.question_id = $1
        ORDER BY RANDOM()
        LIMIT $2
    `, questionID, count-1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answerIDs := []uuid.UUID{correctAnswerID}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		answerIDs = append(answerIDs, id)
	}

	// 3. Shuffle so correct answer isn't always first
	rand.Shuffle(len(answerIDs), func(i, j int) {
		answerIDs[i], answerIDs[j] = answerIDs[j], answerIDs[i]
	})

	if len(answerIDs) < count {
		return nil, fmt.Errorf("question %s has only %d answers, need %d", questionID, len(answerIDs), count)
	}

	return answerIDs, nil
}

func (r *QuizRepo) CountParticipants(ctx context.Context, sessionID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM participants WHERE session_id = $1`, sessionID,
	).Scan(&count)
	return count, err
}

func (r *QuizRepo) CountSubmissionsForQuestion(ctx context.Context, sessionID, questionID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
        SELECT COUNT(*) FROM submissions s
        JOIN participants p ON p.id = s.participant_id
        WHERE p.session_id = $1 AND s.question_id = $2`,
		sessionID, questionID,
	).Scan(&count)
	return count, err
}

// ClearSessionAnswers deletes all submissions for every participant in the
// session and resets their scores to 0. Call before the first question is sent.
func (r *QuizRepo) ClearSessionAnswers(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
        DELETE FROM submissions s
        USING participants p
        WHERE p.id = s.participant_id
          AND p.session_id = $1
    `, sessionID)
	return err
}

var ErrDuplicateSubmission = errors.New("duplicate submission")
