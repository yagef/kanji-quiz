package repository

import (
	"context"
	"fmt"
	"kanji-quiz/server/model"

	"github.com/google/uuid"
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

func (r *QuizRepo) ListAnswerTypes(ctx context.Context) ([]model.AnswerType, error) {
	rows, err := r.db.Query(ctx, `SELECT id, text, title FROM answer_types`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AnswerType
	for rows.Next() {
		var t model.AnswerType
		if err := rows.Scan(&t.ID, &t.Text, &t.Title); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *QuizRepo) ListQuestions(ctx context.Context, quizID uuid.UUID) ([]model.Question, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, quiz_id, type_id, kanji, correct_answer_id FROM questions WHERE quiz_id = $1`, quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Question
	for rows.Next() {
		var q model.Question
		if err := rows.Scan(&q.ID, &q.QuizID, &q.TypeID, &q.Kanji, &q.CorrectAnswerID); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func (r *QuizRepo) CreateQuestion(ctx context.Context, quizID uuid.UUID, typeID int, kanji string) (model.Question, error) {
	var q model.Question
	// correct_answer_id is intentionally omitted — set later via SetCorrectAnswer
	err := r.db.QueryRow(ctx,
		`INSERT INTO questions (quiz_id, type_id, kanji) VALUES ($1, $2, $3)
         RETURNING id, quiz_id, type_id, kanji, correct_answer_id`,
		quizID, typeID, kanji,
	).Scan(&q.ID, &q.QuizID, &q.TypeID, &q.Kanji, &q.CorrectAnswerID)
	return q, err
}

func (r *QuizRepo) GetQuestion(ctx context.Context, id uuid.UUID) (model.Question, error) {
	var q model.Question
	err := r.db.QueryRow(ctx,
		`SELECT id, quiz_id, type_id, kanji, correct_answer_id FROM questions WHERE id = $1`, id,
	).Scan(&q.ID, &q.QuizID, &q.TypeID, &q.Kanji, &q.CorrectAnswerID)
	return q, err
}

func (r *QuizRepo) ListAnswers(ctx context.Context, questionID uuid.UUID) ([]model.Answer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, question_id, text FROM answers WHERE question_id = $1`, questionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Answer
	for rows.Next() {
		var a model.Answer
		if err := rows.Scan(&a.ID, &a.QuestionID, &a.Text); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *QuizRepo) AddAnswer(ctx context.Context, questionID uuid.UUID, text string) (model.Answer, error) {
	var a model.Answer
	err := r.db.QueryRow(ctx,
		`INSERT INTO answers (question_id, text) VALUES ($1, $2) RETURNING id, question_id, text`,
		questionID, text,
	).Scan(&a.ID, &a.QuestionID, &a.Text)
	return a, err
}

// SetCorrectAnswer updates correct_answer_id; the answer must belong to the question
func (r *QuizRepo) SetCorrectAnswer(ctx context.Context, questionID, answerID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE questions SET correct_answer_id = $1
         WHERE id = $2
           AND EXISTS (SELECT 1 FROM answers WHERE id = $1 AND question_id = $2)`,
		answerID, questionID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("answer %s does not belong to question %s", answerID, questionID)
	}
	return nil
}

func (r *QuizRepo) StartSession(ctx context.Context, quizID uuid.UUID) (model.QuizSession, error) {
	var s model.QuizSession
	err := r.db.QueryRow(ctx,
		`INSERT INTO quiz_sessions (quiz_id) VALUES ($1) RETURNING id, quiz_id, token`, quizID,
	).Scan(&s.ID, &s.QuizID, &s.Token)
	return s, err
}
func (r *QuizRepo) DeleteAnswer(ctx context.Context, questionID, answerID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE questions SET correct_answer_id = NULL WHERE id = $1 AND correct_answer_id = $2`, questionID, answerID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM answers WHERE id = $1 AND question_id = $2`, answerID, questionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *QuizRepo) DeleteQuestion(ctx context.Context, quizID, questionID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Break the circular dependency
	_, err = tx.Exec(ctx, `UPDATE questions SET correct_answer_id = NULL WHERE id = $1`, questionID)
	if err != nil {
		return err
	}

	// 2. Delete all answers for this question
	_, err = tx.Exec(ctx, `DELETE FROM answers WHERE question_id = $1`, questionID)
	if err != nil {
		return err
	}

	// 3. Delete the question itself
	_, err = tx.Exec(ctx, `DELETE FROM questions WHERE id = $1 AND quiz_id = $2`, questionID, quizID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
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
