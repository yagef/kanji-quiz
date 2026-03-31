package ws

import (
	"context"
	"errors"
	"testing"
	"time"

	"kanji-quiz/server/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// mockRepo — hand-rolled mock that satisfies repository.QuizRepoIface
// ---------------------------------------------------------------------------

type mockRepo struct {
	// ListQuestions
	questions    []model.Question
	questionsErr error

	// PickRandomAnswersForQuestion
	answerIDs    []uuid.UUID
	answerIDsErr error

	// GetQuestion
	question    model.Question
	questionErr error

	// GetAnswersByIDs
	answers    []model.Answer
	answersErr error

	// ListParticipants
	participants    []model.Participant
	participantsErr error

	// GetParticipantScore
	score    int
	scoreErr error

	// HasParticipantAnswered
	answered    bool
	answeredErr error

	// InsertTimeoutSubmissions
	timeoutErr error

	// EndSession
	endSessionErr error

	// CountParticipants
	participantCount    int
	participantCountErr error

	// CountSubmissionsForQuestion
	submissionCount    int
	submissionCountErr error

	// InsertSubmissionAndUpdateScore
	insertErr error

	// IsAnswerCorrect
	isCorrect    bool
	isCorrectErr error

	// Tracking calls
	endSessionCalled  bool
	timeoutCalled     bool
	insertScoreCalled bool
}

func (m *mockRepo) ListQuestions(_ context.Context, _ uuid.UUID) ([]model.Question, error) {
	return m.questions, m.questionsErr
}
func (m *mockRepo) PickRandomAnswersForQuestion(_ context.Context, _ uuid.UUID, _ int) ([]uuid.UUID, error) {
	return m.answerIDs, m.answerIDsErr
}
func (m *mockRepo) GetQuestion(_ context.Context, _ uuid.UUID) (model.Question, error) {
	return m.question, m.questionErr
}
func (m *mockRepo) GetAnswersByIDs(_ context.Context, _ []uuid.UUID) ([]model.Answer, error) {
	return m.answers, m.answersErr
}
func (m *mockRepo) ListParticipants(_ context.Context, _ uuid.UUID) ([]model.Participant, error) {
	return m.participants, m.participantsErr
}
func (m *mockRepo) GetParticipantScore(_ context.Context, _ uuid.UUID) (int, error) {
	return m.score, m.scoreErr
}
func (m *mockRepo) HasParticipantAnswered(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return m.answered, m.answeredErr
}
func (m *mockRepo) InsertTimeoutSubmissions(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) error {
	m.timeoutCalled = true
	return m.timeoutErr
}
func (m *mockRepo) EndSession(_ context.Context, _ uuid.UUID) error {
	m.endSessionCalled = true
	return m.endSessionErr
}
func (m *mockRepo) CountParticipants(_ context.Context, _ uuid.UUID) (int, error) {
	return m.participantCount, m.participantCountErr
}
func (m *mockRepo) CountSubmissionsForQuestion(_ context.Context, _ uuid.UUID, _ uuid.UUID) (int, error) {
	return m.submissionCount, m.submissionCountErr
}
func (m *mockRepo) InsertSubmissionAndUpdateScore(_ context.Context, _, _, _ uuid.UUID, _ bool, _, _ int) error {
	m.insertScoreCalled = true
	return m.insertErr
}
func (m *mockRepo) IsAnswerCorrect(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isCorrect, m.isCorrectErr
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeQuestions(n int) []model.Question {
	qs := make([]model.Question, n)
	for i := range qs {
		qs[i] = model.Question{ID: uuid.New(), QuizID: uuid.New()}
	}
	return qs
}

func makeAnswerIDs(n int) []uuid.UUID {
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.New()
	}
	return ids
}

func newTestEngine(repo *mockRepo) (*Engine, *Manager) {
	mgr := NewManager()
	eng := NewEngine(repo, mgr)
	return eng, mgr
}

// ---------------------------------------------------------------------------
// NewEngine
// ---------------------------------------------------------------------------

func TestNewEngine(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	require.NotNil(t, eng)
	assert.NotNil(t, eng.sessions)
}

// ---------------------------------------------------------------------------
// InitSession
// ---------------------------------------------------------------------------

func TestEngine_InitSession_Success(t *testing.T) {
	repo := &mockRepo{
		questions: makeQuestions(3),
		answerIDs: makeAnswerIDs(4),
	}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	qID := uuid.New()
	err := eng.InitSession(context.Background(), sID, qID, 15, 5)
	require.NoError(t, err)

	state := eng.getState(sID)
	require.NotNil(t, state)
	assert.Equal(t, PhaseWaiting, state.Phase)
	assert.Equal(t, -1, state.CurrentIndex)
	assert.Len(t, state.Rounds, 3)
	assert.Equal(t, 15*time.Second, state.AnswerDuration)
	assert.Equal(t, 5*time.Second, state.CountdownDuration)
}

func TestEngine_InitSession_NoQuestions(t *testing.T) {
	repo := &mockRepo{questions: nil, questionsErr: nil}
	eng, _ := newTestEngine(repo)

	err := eng.InitSession(context.Background(), uuid.New(), uuid.New(), 15, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no questions")
}

func TestEngine_InitSession_RepoError(t *testing.T) {
	repo := &mockRepo{questionsErr: errors.New("db error")}
	eng, _ := newTestEngine(repo)

	err := eng.InitSession(context.Background(), uuid.New(), uuid.New(), 15, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestEngine_InitSession_AnswerPickError(t *testing.T) {
	repo := &mockRepo{
		questions:    makeQuestions(2),
		answerIDsErr: errors.New("not enough answers"),
	}
	eng, _ := newTestEngine(repo)

	err := eng.InitSession(context.Background(), uuid.New(), uuid.New(), 15, 5)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetPhase
// ---------------------------------------------------------------------------

func TestEngine_GetPhase_UninitializedReturnsWaiting(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	phase := eng.GetPhase(uuid.New())
	assert.Equal(t, PhaseWaiting, phase)
}

func TestEngine_GetPhase_AfterInitIsWaiting(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(2), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 15, 5))
	assert.Equal(t, PhaseWaiting, eng.GetPhase(sID))
}

// ---------------------------------------------------------------------------
// StartQuiz
// ---------------------------------------------------------------------------

func TestEngine_StartQuiz_ErrorWhenNoParticipants(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(2), answerIDs: makeAnswerIDs(4)}
	eng, mgr := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 15, 5))

	// No clients connected → ConnectedCount == 0
	_ = mgr // no clients added
	err := eng.StartQuiz(sID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active participants")
}

func TestEngine_StartQuiz_ErrorWhenNotInitialized(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	err := eng.StartQuiz(uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestEngine_StartQuiz_ErrorWhenNotInWaitingPhase(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(1), answerIDs: makeAnswerIDs(4)}
	eng, mgr := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 1, 1))

	// Force a non-waiting phase
	state := eng.getState(sID)
	state.mu.Lock()
	state.Phase = PhasePaused
	state.mu.Unlock()

	// Add a connected client so ConnectedCount > 0
	h := mgr.GetOrCreate(sID)
	h.AddClient(newTestClient(sID, uuid.New()))

	err := eng.StartQuiz(sID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start from phase")
}

// ---------------------------------------------------------------------------
// NextQuestion
// ---------------------------------------------------------------------------

func TestEngine_NextQuestion_ErrorWhenSessionNotFound(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	err := eng.NextQuestion(uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestEngine_NextQuestion_ErrorWhenNotPaused(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(3), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 15, 5))
	// Phase is waiting, not paused
	err := eng.NextQuestion(sID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can only go next from paused phase")
}

func TestEngine_NextQuestion_AdvancesIndex(t *testing.T) {
	repo := &mockRepo{
		questions:        makeQuestions(3),
		answerIDs:        makeAnswerIDs(4),
		participantCount: 1,
		participants:     []model.Participant{{ID: uuid.New()}},
	}
	eng, mgr := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 1))

	state := eng.getState(sID)

	// Simulate: currently at index 0, paused
	h := mgr.GetOrCreate(sID)
	h.AddClient(newTestClient(sID, uuid.New()))
	state.mu.Lock()
	state.Phase = PhasePaused
	state.CurrentIndex = 0
	state.mu.Unlock()

	err := eng.NextQuestion(sID)
	require.NoError(t, err)

	// Give the goroutine a brief moment to transition to countdown, then
	// just check the index advanced — we don't wait for the full question run.
	time.Sleep(20 * time.Millisecond)
	state.mu.RLock()
	idx := state.CurrentIndex
	state.mu.RUnlock()
	assert.Equal(t, 1, idx)
}

func TestEngine_NextQuestion_LastQuestion_TransitionsToFinished(t *testing.T) {
	repo := &mockRepo{
		questions:    makeQuestions(2),
		answerIDs:    makeAnswerIDs(4),
		participants: []model.Participant{{ID: uuid.New()}},
	}
	eng, mgr := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 1))

	state := eng.getState(sID)
	h := mgr.GetOrCreate(sID)
	h.AddClient(newTestClient(sID, uuid.New()))

	// Manually set to last index and paused phase
	state.mu.Lock()
	state.Phase = PhasePaused
	state.CurrentIndex = 1 // last valid index for 2 questions
	state.mu.Unlock()

	err := eng.NextQuestion(sID)
	require.NoError(t, err)

	// Allow async goroutines to settle
	time.Sleep(20 * time.Millisecond)

	state.mu.RLock()
	phase := state.Phase
	state.mu.RUnlock()
	assert.Equal(t, PhaseFinished, phase)
	assert.True(t, repo.endSessionCalled, "EndSession should have been called")
}

// ---------------------------------------------------------------------------
// CanAnswer
// ---------------------------------------------------------------------------

func TestEngine_CanAnswer_FalseWhenSessionMissing(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	_, _, ok := eng.CanAnswer(uuid.New(), uuid.New())
	assert.False(t, ok)
}

func TestEngine_CanAnswer_FalseWhenNotAnsweringPhase(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(1), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	qID := makeQuestions(1)[0].ID
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 15, 5))

	_, _, ok := eng.CanAnswer(sID, qID)
	assert.False(t, ok)
}

func TestEngine_CanAnswer_FalseWhenWrongQuestion(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(2), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 5))

	state := eng.getState(sID)
	state.mu.Lock()
	state.Phase = PhaseAnswering
	state.CurrentIndex = 0
	state.Rounds[0].Deadline = time.Now().Add(60 * time.Second)
	state.mu.Unlock()

	_, _, ok := eng.CanAnswer(sID, uuid.New()) // wrong question ID
	assert.False(t, ok)
}

func TestEngine_CanAnswer_FalseWhenDeadlinePassed(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(1), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 5))

	state := eng.getState(sID)
	state.mu.Lock()
	state.Phase = PhaseAnswering
	state.CurrentIndex = 0
	qID := state.Rounds[0].QuestionID
	state.Rounds[0].Deadline = time.Now().Add(-1 * time.Second) // already expired
	state.mu.Unlock()

	_, _, ok := eng.CanAnswer(sID, qID)
	assert.False(t, ok)
}

func TestEngine_CanAnswer_TrueWhenValid(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(1), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 5))

	state := eng.getState(sID)
	state.mu.Lock()
	state.Phase = PhaseAnswering
	state.CurrentIndex = 0
	qID := state.Rounds[0].QuestionID
	state.Rounds[0].Deadline = time.Now().Add(60 * time.Second)
	state.mu.Unlock()

	gotState, round, ok := eng.CanAnswer(sID, qID)
	assert.True(t, ok)
	assert.NotNil(t, gotState)
	assert.NotNil(t, round)
	assert.Equal(t, qID, round.QuestionID)
}

// ---------------------------------------------------------------------------
// SessionState.CurrentRound
// ---------------------------------------------------------------------------

func TestSessionState_CurrentRound_NilWhenEmpty(t *testing.T) {
	s := &SessionState{CurrentIndex: -1}
	assert.Nil(t, s.CurrentRound())
}

func TestSessionState_CurrentRound_NilWhenIndexOutOfRange(t *testing.T) {
	s := &SessionState{
		Rounds:       make([]QuestionRound, 2),
		CurrentIndex: 5,
	}
	assert.Nil(t, s.CurrentRound())
}

func TestSessionState_CurrentRound_ReturnsCorrectRound(t *testing.T) {
	qID := uuid.New()
	s := &SessionState{
		Rounds: []QuestionRound{
			{QuestionID: uuid.New()},
			{QuestionID: qID},
		},
		CurrentIndex: 1,
	}
	r := s.CurrentRound()
	require.NotNil(t, r)
	assert.Equal(t, qID, r.QuestionID)
}

// ---------------------------------------------------------------------------
// NotifyAnswerSubmitted
// ---------------------------------------------------------------------------

func TestEngine_NotifyAnswerSubmitted_NoopWhenSessionMissing(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	// Should not panic
	assert.NotPanics(t, func() {
		eng.NotifyAnswerSubmitted(context.Background(), uuid.New())
	})
}

func TestEngine_NotifyAnswerSubmitted_NoopWhenNotAnsweringPhase(t *testing.T) {
	repo := &mockRepo{questions: makeQuestions(2), answerIDs: makeAnswerIDs(4)}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 5))
	// Phase is waiting — signal must be silently ignored
	assert.NotPanics(t, func() {
		eng.NotifyAnswerSubmitted(context.Background(), sID)
	})
}

func TestEngine_NotifyAnswerSubmitted_SignalsWhenAllAnswered(t *testing.T) {
	repo := &mockRepo{
		questions:        makeQuestions(1),
		answerIDs:        makeAnswerIDs(4),
		participantCount: 2,
		submissionCount:  2, // everyone has answered
	}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 5))

	state := eng.getState(sID)
	ch := make(chan struct{}, 1)
	state.mu.Lock()
	state.Phase = PhaseAnswering
	state.CurrentIndex = 0
	state.allAnsweredCh = ch
	state.mu.Unlock()

	eng.NotifyAnswerSubmitted(context.Background(), sID)

	select {
	case <-ch:
		// expected
	default:
		t.Fatal("allAnsweredCh was not signaled")
	}
}

func TestEngine_NotifyAnswerSubmitted_NoSignalWhenNotAllAnswered(t *testing.T) {
	repo := &mockRepo{
		questions:        makeQuestions(1),
		answerIDs:        makeAnswerIDs(4),
		participantCount: 3,
		submissionCount:  1, // only 1 of 3 answered
	}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 60, 5))

	state := eng.getState(sID)
	ch := make(chan struct{}, 1)
	state.mu.Lock()
	state.Phase = PhaseAnswering
	state.CurrentIndex = 0
	state.allAnsweredCh = ch
	state.mu.Unlock()

	eng.NotifyAnswerSubmitted(context.Background(), sID)

	select {
	case <-ch:
		t.Fatal("channel should NOT have been signaled yet")
	default:
		// correct — not all answered
	}
}

// ---------------------------------------------------------------------------
// setState / getState
// ---------------------------------------------------------------------------

func TestEngine_SetGetState(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})

	sID := uuid.New()
	s := &SessionState{SessionID: sID, Phase: PhasePaused}
	eng.setState(s)

	got := eng.getState(sID)
	require.NotNil(t, got)
	assert.Equal(t, PhasePaused, got.Phase)
}

func TestEngine_GetState_NilForUnknown(t *testing.T) {
	eng, _ := newTestEngine(&mockRepo{})
	assert.Nil(t, eng.getState(uuid.New()))
}

// ---------------------------------------------------------------------------
// InitSession: rounds have correct structure
// ---------------------------------------------------------------------------

func TestEngine_InitSession_RoundsHaveAnswerIDs(t *testing.T) {
	aIDs := makeAnswerIDs(4)
	repo := &mockRepo{
		questions: makeQuestions(2),
		answerIDs: aIDs,
	}
	eng, _ := newTestEngine(repo)

	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 15, 5))

	state := eng.getState(sID)
	for _, r := range state.Rounds {
		assert.Equal(t, aIDs, r.AnswerIDs)
	}
}

func TestEngine_InitSession_IndexesAreZeroBased(t *testing.T) {
	repo := &mockRepo{
		questions: makeQuestions(4),
		answerIDs: makeAnswerIDs(4),
	}
	eng, _ := newTestEngine(repo)
	sID := uuid.New()
	require.NoError(t, eng.InitSession(context.Background(), sID, uuid.New(), 15, 5))

	state := eng.getState(sID)
	for i, r := range state.Rounds {
		assert.Equal(t, i, r.Index, "round index should match slice position")
	}
}
