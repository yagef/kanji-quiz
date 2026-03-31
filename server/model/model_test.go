package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Quiz
// ---------------------------------------------------------------------------

func TestQuiz_Fields(t *testing.T) {
	id := uuid.New()
	q := Quiz{ID: id, Title: "N5 Kanji"}
	assert.Equal(t, id, q.ID)
	assert.Equal(t, "N5 Kanji", q.Title)
}

// ---------------------------------------------------------------------------
// AnswerType
// ---------------------------------------------------------------------------

func TestAnswerType_Fields(t *testing.T) {
	at := AnswerType{ID: 3, Text: "reading", Title: "Reading"}
	assert.Equal(t, 3, at.ID)
	assert.Equal(t, "reading", at.Text)
	assert.Equal(t, "Reading", at.Title)
}

// ---------------------------------------------------------------------------
// Question
// ---------------------------------------------------------------------------

func TestQuestion_NullableCorrectAnswerID(t *testing.T) {
	// CorrectAnswerID is a pointer and should be nil-able
	q := Question{ID: uuid.New()}
	assert.Nil(t, q.CorrectAnswerID, "CorrectAnswerID should default to nil")

	aID := uuid.New()
	q.CorrectAnswerID = &aID
	assert.Equal(t, &aID, q.CorrectAnswerID)
}

func TestQuestion_Fields(t *testing.T) {
	qID := uuid.New()
	quizID := uuid.New()
	aID := uuid.New()
	q := Question{
		ID:              qID,
		QuizID:          quizID,
		TypeID:          2,
		TypeText:        "意味",
		Kanji:           "日",
		CorrectAnswerID: &aID,
	}
	assert.Equal(t, qID, q.ID)
	assert.Equal(t, quizID, q.QuizID)
	assert.Equal(t, 2, q.TypeID)
	assert.Equal(t, "意味", q.TypeText)
	assert.Equal(t, "日", q.Kanji)
	assert.Equal(t, aID, *q.CorrectAnswerID)
}

// ---------------------------------------------------------------------------
// Answer
// ---------------------------------------------------------------------------

func TestAnswer_Fields(t *testing.T) {
	aID := uuid.New()
	qID := uuid.New()
	a := Answer{ID: aID, QuestionID: qID, Text: "にち"}
	assert.Equal(t, aID, a.ID)
	assert.Equal(t, qID, a.QuestionID)
	assert.Equal(t, "にち", a.Text)
}

// ---------------------------------------------------------------------------
// QuizSession
// ---------------------------------------------------------------------------

func TestQuizSession_NullableTimestamps(t *testing.T) {
	s := QuizSession{ID: uuid.New(), QuizID: uuid.New()}
	assert.Nil(t, s.StartedAt, "StartedAt should default to nil")
	assert.Nil(t, s.EndedAt, "EndedAt should default to nil")
}

func TestQuizSession_WithTimestamps(t *testing.T) {
	now := time.Now()
	end := now.Add(10 * time.Minute)
	s := QuizSession{
		ID:        uuid.New(),
		QuizID:    uuid.New(),
		StartedAt: &now,
		EndedAt:   &end,
	}
	assert.Equal(t, now, *s.StartedAt)
	assert.Equal(t, end, *s.EndedAt)
}

// ---------------------------------------------------------------------------
// Participant
// ---------------------------------------------------------------------------

func TestParticipant_Fields(t *testing.T) {
	p := Participant{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Name:      "Alice",
		Score:     850,
		SessionID: uuid.New(),
	}
	assert.Equal(t, "Alice", p.Name)
	assert.Equal(t, 850, p.Score)
}

func TestParticipant_ZeroScore(t *testing.T) {
	p := Participant{ID: uuid.New()}
	assert.Equal(t, 0, p.Score)
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

func TestUser_Fields(t *testing.T) {
	id := uuid.New()
	u := User{ID: id, Name: "Bob"}
	assert.Equal(t, id, u.ID)
	assert.Equal(t, "Bob", u.Name)
}

// ---------------------------------------------------------------------------
// HistoryEntry
// ---------------------------------------------------------------------------

func TestHistoryEntry_Fields(t *testing.T) {
	start := time.Now()
	end := start.Add(30 * time.Minute)
	e := HistoryEntry{
		ParticipantID: uuid.New(),
		SessionID:     uuid.New(),
		QuizTitle:     "JLPT N4",
		StartedAt:     start,
		EndedAt:       &end,
		Score:         1200,
	}
	assert.Equal(t, "JLPT N4", e.QuizTitle)
	assert.Equal(t, 1200, e.Score)
	assert.Equal(t, start, e.StartedAt)
	assert.Equal(t, end, *e.EndedAt)
}

func TestHistoryEntry_NullableEndedAt(t *testing.T) {
	e := HistoryEntry{StartedAt: time.Now()}
	assert.Nil(t, e.EndedAt)
}

// ---------------------------------------------------------------------------
// SubmissionDetail
// ---------------------------------------------------------------------------

func TestSubmissionDetail_Fields(t *testing.T) {
	d := SubmissionDetail{
		Kanji:         "木",
		QuestionType:  "読み方",
		AnswerGiven:   "き",
		CorrectAnswer: "き",
		IsCorrect:     true,
		TimeTaken:     2500 * time.Millisecond,
	}
	assert.Equal(t, "木", d.Kanji)
	assert.True(t, d.IsCorrect)
	assert.Equal(t, 2500*time.Millisecond, d.TimeTaken)
}

func TestSubmissionDetail_EmptyAnswerGiven(t *testing.T) {
	d := SubmissionDetail{
		Kanji:         "火",
		AnswerGiven:   "", // timed out — no answer
		CorrectAnswer: "ひ",
		IsCorrect:     false,
	}
	assert.Empty(t, d.AnswerGiven)
	assert.False(t, d.IsCorrect)
}

// ---------------------------------------------------------------------------
// QuestionWithAnswers
// ---------------------------------------------------------------------------

func TestQuestionWithAnswers_Composition(t *testing.T) {
	q := Question{ID: uuid.New(), Kanji: "山"}
	answers := []Answer{
		{ID: uuid.New(), Text: "やま"},
		{ID: uuid.New(), Text: "かわ"},
	}
	qwa := QuestionWithAnswers{Question: q, Answers: answers}
	assert.Equal(t, "山", qwa.Question.Kanji)
	assert.Len(t, qwa.Answers, 2)
}

func TestQuestionWithAnswers_EmptyAnswers(t *testing.T) {
	qwa := QuestionWithAnswers{Question: Question{ID: uuid.New()}}
	assert.Nil(t, qwa.Answers)
}

// ---------------------------------------------------------------------------
// SessionState (model package)
// ---------------------------------------------------------------------------

func TestSessionState_PhaseConstants(t *testing.T) {
	assert.Equal(t, Phase("waiting"), PhaseWaiting)
	assert.Equal(t, Phase("countdown"), PhaseCountdown)
	assert.Equal(t, Phase("answering"), PhaseAnswering)
	assert.Equal(t, Phase("paused"), PhasePaused)
	assert.Equal(t, Phase("finished"), PhaseFinished)
}

func TestSessionState_InitialState(t *testing.T) {
	sID := uuid.New()
	qID := uuid.New()
	q1 := SessionQuestion{QuestionID: uuid.New(), AnswerIDs: []uuid.UUID{uuid.New(), uuid.New()}}
	q2 := SessionQuestion{QuestionID: uuid.New(), AnswerIDs: []uuid.UUID{uuid.New(), uuid.New()}}

	ss := SessionState{
		SessionID:        sID,
		QuizID:           qID,
		Phase:            PhaseWaiting,
		CurrentIndex:     -1,
		Questions:        []SessionQuestion{q1, q2},
		QuestionDuration: 15,
	}

	assert.Equal(t, sID, ss.SessionID)
	assert.Equal(t, PhaseWaiting, ss.Phase)
	assert.Equal(t, -1, ss.CurrentIndex)
	assert.Len(t, ss.Questions, 2)
	assert.Equal(t, 15, ss.QuestionDuration)
}

func TestSessionQuestion_AnswerIDs(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	sq := SessionQuestion{
		QuestionID: uuid.New(),
		AnswerIDs:  ids,
	}
	assert.Len(t, sq.AnswerIDs, 4)
	for i, id := range sq.AnswerIDs {
		assert.Equal(t, ids[i], id)
	}
}
