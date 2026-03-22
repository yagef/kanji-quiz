package repository

import (
	"context"
	"errors"
	"kanji-quiz/server/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuizRepo struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *QuizRepo { return &QuizRepo{db: db} }

func (r *QuizRepo) ListQuizzes(ctx context.Context) ([]model.Quiz, error) {
	rows, err := r.db.Query(ctx, `SELECT id, title FROM quizzes ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Quiz
	for rows.Next() {
		var q model.Quiz
		if err := rows.Scan(&q.ID, &q.Title); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func (r *QuizRepo) CreateQuiz(ctx context.Context, title string) (model.Quiz, error) {
	var q model.Quiz
	err := r.db.QueryRow(ctx,
		`INSERT INTO quizzes (title) VALUES ($1) RETURNING id, title`, title,
	).Scan(&q.ID, &q.Title)
	return q, err
}

func (r *QuizRepo) GetQuiz(ctx context.Context, id uuid.UUID) (model.Quiz, error) {
	var q model.Quiz
	err := r.db.QueryRow(ctx, `SELECT id, title FROM quizzes WHERE id = $1`, id).Scan(&q.ID, &q.Title)
	return q, err
}

func (r *QuizRepo) DeleteQuiz(ctx context.Context, quizID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Note: This hard-delete assumes there are no active quiz_sessions/submissions yet.
	// If submissions exist, deleting the quiz will fail due to FK constraints on those tables.

	// 1. Break circular dependencies for all questions in this quiz
	_, err = tx.Exec(ctx, `UPDATE questions SET correct_answer_id = NULL WHERE quiz_id = $1`, quizID)
	if err != nil {
		return err
	}

	// 2. Delete all answers for all questions in this quiz
	_, err = tx.Exec(ctx, `
		DELETE FROM answers 
		WHERE question_id IN (SELECT id FROM questions WHERE quiz_id = $1)
	`, quizID)
	if err != nil {
		return err
	}

	// 3. Delete the questions
	_, err = tx.Exec(ctx, `DELETE FROM questions WHERE quiz_id = $1`, quizID)
	if err != nil {
		return err
	}

	// 4. Delete the quiz
	_, err = tx.Exec(ctx, `DELETE FROM quizzes WHERE id = $1`, quizID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *QuizRepo) GetQuestionsWithAnswers(ctx context.Context, quizID uuid.UUID) ([]model.QuestionWithAnswers, error) {
	rows, err := r.db.Query(ctx, `
		SELECT q.id, q.quiz_id, q.type_id, q.kanji, q.correct_answer_id,
		       a.id, a.question_id, a.text
		FROM questions q
		JOIN answers a ON a.question_id = q.id
		WHERE q.quiz_id = $1
		ORDER BY q.id
	`, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := map[uuid.UUID]model.QuestionWithAnswers{}
	var order []uuid.UUID

	for rows.Next() {
		var q model.Question
		var a model.Answer
		if err := rows.Scan(&q.ID, &q.QuizID, &q.TypeID, &q.Kanji, &q.CorrectAnswerID,
			&a.ID, &a.QuestionID, &a.Text); err != nil {
			return nil, err
		}
		qwa, ok := m[q.ID]
		if !ok {
			qwa = model.QuestionWithAnswers{Question: q}
			order = append(order, q.ID)
		}
		qwa.Answers = append(qwa.Answers, a)
		m[q.ID] = qwa
	}

	out := make([]model.QuestionWithAnswers, 0, len(order))
	for _, id := range order {
		out = append(out, m[id])
	}
	return out, nil
}

func (r *QuizRepo) CreateSubmission(ctx context.Context, participantID, questionID, answerID uuid.UUID, correct bool, time int) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO submissions (participant_id, question_id, answer_id, is_correct, time_taken_ms)
		VALUES ($1, $2, $3, $4, $5)
	`, participantID, questionID, answerID, correct, time)
	return err
}

// IsAnswerCorrect checks if given answer is the correct one for the question.
func (r *QuizRepo) IsAnswerCorrect(ctx context.Context, questionID, answerID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM questions q
		WHERE q.id = $1
		  AND q.correct_answer_id = $2
	`, questionID, answerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

var maxScore = 1000
var bonusThreshold = 3000

// InsertSubmissionAndUpdateScore inserts into submissions and updates participant score.
func (r *QuizRepo) InsertSubmissionAndUpdateScore(
	ctx context.Context,
	participantID, questionID, answerID uuid.UUID,
	isCorrect bool,
	timeTakenMs int,
	timeLimit int,
) error {
	var score int
	if isCorrect {
		if timeTakenMs < bonusThreshold {
			score = maxScore
		} else {
			score = maxScore * (timeLimit - timeTakenMs) / (timeLimit - bonusThreshold)
		}
		if score < 0 {
			score = 0
		}
	}

	_, err := r.db.Exec(ctx, `
        INSERT INTO submissions (participant_id, question_id, answer_id, is_correct, time_taken_ms, score)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, participantID, questionID, answerID, isCorrect, timeTakenMs, score)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateSubmission
		}
		return err
	}
	return nil
}

func (r *QuizRepo) GetParticipantScore(ctx context.Context, participantID uuid.UUID) (int, error) {
	var score int
	err := r.db.QueryRow(ctx, `
        SELECT COALESCE(SUM(score), 0)
        FROM submissions
        WHERE participant_id = $1
    `, participantID).Scan(&score)
	return score, err
}

func (r *QuizRepo) HasParticipantAnswered(ctx context.Context, participantID, questionID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM submissions
		WHERE participant_id = $1
		  AND question_id   = $2
	`, participantID, questionID).Scan(&count)
	return count > 0, err
}

func (r *QuizRepo) InsertTimeoutSubmissions(
	ctx context.Context,
	sessionID, questionID uuid.UUID,
	timeLimitMs int,
) error {
	_, err := r.db.Exec(ctx, `
    INSERT INTO submissions (participant_id, question_id, answer_id, is_correct, time_taken_ms, score)
    SELECT p.id, $2, NULL, false, $3, 0
    FROM participants p
    WHERE p.session_id = $1
      AND NOT EXISTS (
          SELECT 1 FROM submissions s
          WHERE s.participant_id = p.id AND s.question_id = $2
      )
    ON CONFLICT DO NOTHING
    `, sessionID, questionID, timeLimitMs)
	return err
}
