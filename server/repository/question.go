package repository

import (
	"context"
	"fmt"
	"kanji-quiz/server/model"

	"github.com/google/uuid"
)

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
		`SELECT q.id, q.quiz_id, q.type_id, q.kanji, q.correct_answer_id, a.text
				FROM questions AS q
				INNER JOIN public.answer_types a on q.type_id = a.id
                WHERE q.quiz_id = $1`, quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Question
	for rows.Next() {
		var q model.Question
		if err := rows.Scan(&q.ID, &q.QuizID, &q.TypeID, &q.Kanji, &q.CorrectAnswerID, &q.TypeText); err != nil {
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
		`SELECT q.id, q.quiz_id, q.type_id, q.kanji, q.correct_answer_id, a.text
			FROM questions AS q
			INNER JOIN answer_types a on q.type_id = a.id
			WHERE q.id = $1`, id,
	).Scan(&q.ID, &q.QuizID, &q.TypeID, &q.Kanji, &q.CorrectAnswerID, &q.TypeText)
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

func (r *QuizRepo) UpdateQuestion(ctx context.Context, questionID uuid.UUID, kanji string, typeID int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE questions SET kanji = $1, type_id = $2 WHERE id = $3`,
		kanji, typeID, questionID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("question not found")
	}
	return nil
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

func (r *QuizRepo) GetAnswersByIDs(ctx context.Context, answerIDs []uuid.UUID) ([]model.Answer, error) {
	rows, err := r.db.Query(ctx,
		`SELECT a.id, a.question_id, a.text
FROM answers a
JOIN unnest($1::uuid[]) WITH ORDINALITY AS t(id, ord) ON a.id = t.id
ORDER BY t.ord`, answerIDs,
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
