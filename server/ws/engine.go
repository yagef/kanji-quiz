package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"kanji-quiz/server/repository"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (s *SessionState) CurrentRound() *QuestionRound {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.CurrentIndex < 0 || s.CurrentIndex >= len(s.Rounds) {
		return nil
	}
	r := s.Rounds[s.CurrentIndex]
	return &r
}

type Engine struct {
	repo     *repository.QuizRepo
	hubs     *Manager
	sessions map[uuid.UUID]*SessionState

	mu sync.RWMutex
}

func NewEngine(repo *repository.QuizRepo, hubs *Manager) *Engine {
	return &Engine{
		repo:     repo,
		hubs:     hubs,
		sessions: make(map[uuid.UUID]*SessionState),
	}
}

func (e *Engine) getState(sessionID uuid.UUID) *SessionState {
	e.mu.RLock()
	s := e.sessions[sessionID]
	e.mu.RUnlock()
	return s
}

func (e *Engine) setState(s *SessionState) {
	e.mu.Lock()
	e.sessions[s.SessionID] = s
	e.mu.Unlock()
}

func (e *Engine) InitSession(ctx context.Context, sessionID, quizID uuid.UUID, answerSeconds, countdownSeconds int) error {
	questions, err := e.repo.ListQuestions(ctx, quizID)
	if err != nil {
		return err
	}
	if len(questions) == 0 {
		return fmt.Errorf("no questions for quiz")
	}

	rand.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})

	rounds := make([]QuestionRound, len(questions))
	for i, q := range questions {
		answerIDs, err := e.repo.PickRandomAnswersForQuestion(ctx, q.ID, 4)
		if err != nil {
			return err
		}
		rounds[i] = QuestionRound{
			QuestionID: q.ID,
			AnswerIDs:  answerIDs,
			Index:      i,
		}
	}

	state := &SessionState{
		SessionID:         sessionID,
		QuizID:            quizID,
		Phase:             PhaseWaiting,
		Rounds:            rounds,
		CurrentIndex:      -1,
		CountdownDuration: time.Duration(countdownSeconds) * time.Second,
		AnswerDuration:    time.Duration(answerSeconds) * time.Second,
	}
	e.setState(state)

	return nil
}

func (e *Engine) StartQuiz(ctx context.Context, sessionID uuid.UUID) error {
	state := e.getState(sessionID)
	if state == nil {
		return fmt.Errorf("session not initialized")
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Phase != PhaseWaiting {
		return fmt.Errorf("cannot start from phase %s", state.Phase)
	}

	state.CurrentIndex = 0
	go e.runQuestion(ctx, state.SessionID, state.CurrentIndex)
	return nil
}

func (e *Engine) NextQuestion(ctx context.Context, sessionID uuid.UUID) error {
	state := e.getState(sessionID)
	if state == nil {
		return fmt.Errorf("session not found")
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Phase != PhasePaused {
		return fmt.Errorf("can only go next from paused phase")
	}

	if state.CurrentIndex+1 >= len(state.Rounds) {
		// No more questions
		state.Phase = PhaseFinished
		go e.broadcastState(ctx, state.SessionID)
		return nil
	}

	state.CurrentIndex++
	go e.runQuestion(ctx, state.SessionID, state.CurrentIndex)
	return nil
}

func (e *Engine) runQuestion(ctx context.Context, sessionID uuid.UUID, index int) {
	state := e.getState(sessionID)
	if state == nil {
		return
	}

	// 1) COUNTDOWN PHASE
	state.mu.Lock()
	if index < 0 || index >= len(state.Rounds) {
		state.mu.Unlock()
		return
	}

	state.Phase = PhaseCountdown
	state.CountdownDeadline = time.Now().Add(state.CountdownDuration)
	// No deadline yet, just countdown duration
	state.mu.Unlock()

	// Broadcast "countdown starting"
	_ = e.broadcastState(ctx, sessionID)

	// Tick every second so clients get live remaining_seconds updates
	ticker := time.NewTicker(time.Second)
	timer := time.NewTimer(state.CountdownDuration)
loop:
	for {
		select {
		case <-ticker.C:
			_ = e.broadcastState(ctx, sessionID)
		case <-timer.C:
			ticker.Stop()
			break loop
		}
	}

	// 2) ANSWERING PHASE
	state.mu.Lock()
	if state.CurrentIndex != index || state.Phase != PhaseCountdown {
		// Admin might have cancelled or moved; don't continue this question
		state.mu.Unlock()
		return
	}

	state.Phase = PhaseAnswering
	now := time.Now()
	r := &state.Rounds[index]
	r.Deadline = now.Add(state.AnswerDuration)
	state.Rounds[index] = *r // write back
	state.mu.Unlock()

	// Broadcast "answering started" with question/answers/deadline
	_ = e.broadcastState(ctx, sessionID)

	// 3) WAIT for answer duration
	time.Sleep(state.AnswerDuration)

	// 4) PAUSED PHASE
	state.mu.Lock()
	if state.CurrentIndex == index && state.Phase == PhaseAnswering {
		state.Phase = PhasePaused
	}
	state.mu.Unlock()

	_ = e.broadcastState(ctx, sessionID)
}

func (e *Engine) broadcastState(ctx context.Context, sessionID uuid.UUID) error {
	base, err := e.buildBaseStatePayload(ctx, sessionID)
	if err != nil {
		return err
	}

	rawPayload, err := json.Marshal(base)
	if err != nil {
		return err
	}
	env := Envelope{Type: MsgStateSync, Payload: rawPayload}
	msg, err := json.Marshal(env)
	if err != nil {
		return err
	}

	hub := e.hubs.GetOrCreate(sessionID)
	hub.Broadcast(msg)
	return nil
}

// CanAnswer returns whether the session is in answering phase and before deadline.
func (e *Engine) CanAnswer(sessionID, questionID uuid.UUID) (*SessionState, *QuestionRound, bool) {
	state := e.getState(sessionID)
	if state == nil {
		return nil, nil, false
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	if state.Phase != PhaseAnswering {
		return state, nil, false
	}
	if state.CurrentIndex < 0 || state.CurrentIndex >= len(state.Rounds) {
		return state, nil, false
	}

	r := state.Rounds[state.CurrentIndex]
	if r.QuestionID != questionID {
		return state, nil, false
	}

	if time.Now().After(r.Deadline) {
		return state, &r, false
	}

	return state, &r, true
}

// internal/game/engine_state.go
func (e *Engine) buildBaseStatePayload(ctx context.Context, sessionID uuid.UUID) (StateSyncPayload, error) {
	state := e.getState(sessionID)
	if state == nil {
		return StateSyncPayload{}, nil
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	phase := state.Phase
	var (
		questionID   string
		questionText string
		answersVO    []AnswerVO
		remainingSec int
		totalSec     int
		idx          int
		totalQ       = len(state.Rounds)
	)

	if state.CurrentIndex >= 0 && state.CurrentIndex < len(state.Rounds) {
		r := state.Rounds[state.CurrentIndex]
		idx = r.Index + 1

		if state.Phase == PhaseAnswering {
			q, err := e.repo.GetQuestion(ctx, r.QuestionID)
			if err == nil {
				questionID = q.ID.String()
				questionText = q.Kanji
			}
			answerRows, err := e.repo.GetAnswersByIDs(ctx, r.AnswerIDs)
			if err == nil {
				for _, a := range answerRows {
					answersVO = append(answersVO, AnswerVO{
						ID:   a.ID.String(),
						Text: a.Text,
					})
				}
			}
			now := time.Now()
			if now.Before(r.Deadline) {
				remainingSec = int(r.Deadline.Sub(now).Seconds()) + 1
			}
			totalSec = int(state.AnswerDuration.Seconds())
		}
	}

	return StateSyncPayload{
		Phase:            phase,
		QuestionID:       questionID,
		QuestionText:     questionText,
		Answers:          answersVO,
		RemainingSeconds: remainingSec,
		TotalSeconds:     totalSec,
		QuestionIndex:    idx,
		TotalQuestions:   totalQ,
		Score:            0, // will be filled per participant
	}, nil
}

func (e *Engine) BroadcastStateToParticipant(ctx context.Context, sessionID, participantID uuid.UUID) error {
	base, err := e.buildBaseStatePayload(ctx, sessionID)
	if err != nil {
		return err
	}

	// Fill score for this participant
	score, err := e.repo.GetParticipantScore(ctx, participantID)
	if err == nil {
		base.Score = score
	}

	if base.QuestionID != "" {
		questionID, err := uuid.Parse(base.QuestionID)
		if err == nil {
			answered, err := e.repo.HasParticipantAnswered(ctx, participantID, questionID)
			if err == nil {
				base.HasAnsweredCurrent = answered
			}
		}
	}

	rawPayload, err := json.Marshal(base)
	if err != nil {
		return err
	}
	env := Envelope{Type: MsgStateSync, Payload: rawPayload}
	msg, err := json.Marshal(env)
	if err != nil {
		return err
	}

	hub := e.hubs.GetOrCreate(sessionID)
	hub.SendToParticipant(participantID, msg)
	return nil
}

func (e *Engine) GetPhase(sessionID uuid.UUID) StatePhase {
	state := e.getState(sessionID)
	if state == nil {
		return PhaseWaiting // not initialized yet = waiting
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.Phase
}
