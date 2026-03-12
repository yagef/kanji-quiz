package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"kanji-quiz/server/model"
	"kanji-quiz/server/repository"
	"kanji-quiz/server/ws"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // TODO: tighten in prod
}

type WSHandler struct {
	repo    *repository.QuizRepo
	manager *ws.Manager
	engine  *ws.Engine
}

func NewWSHandler(repo *repository.QuizRepo, manager *ws.Manager, engine *ws.Engine) *WSHandler {
	return &WSHandler{repo: repo, manager: manager, engine: engine}
}

func (h *WSHandler) ParticipantWS(c *gin.Context) {
	participantID, ok := mustUUID(c, c.Param("participantID"))
	if !ok {
		return
	}
	// Load participant to get sessionID and validate
	p, err := h.repo.GetParticipant(c.Request.Context(), participantID)
	if err != nil {
		c.String(http.StatusNotFound, "participant not found")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Conn:          conn,
		Send:          make(chan []byte, 16),
		SessionID:     p.SessionID,
		ParticipantID: participantID,
	}
	hub := h.manager.GetOrCreate(p.SessionID)
	hub.AddClient(client)

	go writePump(hub, client)
	// Send current state immediately on connect so the client
	// has the right phase and hasAnsweredCurrent from the start.
	go func() {
		_ = h.engine.BroadcastStateToParticipant(
			context.Background(),
			p.SessionID,
			participantID,
		)
	}()
	go h.readPump(hub, client, &p)
}

func writePump(hub *ws.SessionHub, c *ws.Client) {
	defer func() {
		hub.RemoveClient(c.ParticipantID)
		_ = c.Conn.Close()
	}()

	for {
		msg, ok := <-c.Send
		if !ok {
			// Channel closed by hub — send close frame
			_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (h *WSHandler) readPump(hub *ws.SessionHub, c *ws.Client, p *model.Participant) {
	defer func() {
		hub.RemoveClient(c.ParticipantID)
		_ = c.Conn.Close()
	}()

	// Reasonable limits to prevent slow/malicious clients
	c.Conn.SetReadLimit(512)
	err := c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			// Client disconnected or error — exit quietly
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				// log if you want: log.Printf("ws read error: %v", err)
			}
			return
		}

		var env ws.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue // ignore malformed messages
		}

		switch env.Type {
		case ws.MsgAnswer:
			var payload ws.AnswerPayload
			if err := json.Unmarshal(env.Payload, &payload); err != nil {
				h.sendError(c, "malformed answer payload")
				continue
			}
			h.handleAnswer(c, p, payload)

		case ws.MsgHeartbeat:
			// Reset deadline on any heartbeat from client
			_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
	}
}

func (h *WSHandler) handleAnswer(c *ws.Client, p *model.Participant, payload ws.AnswerPayload) {
	ctx := context.Background()

	questionID, err := uuid.Parse(payload.QuestionID)
	if err != nil {
		h.sendError(c, "invalid question id")
		return
	}
	answerID, err := uuid.Parse(payload.AnswerID)
	if err != nil {
		h.sendError(c, "invalid answer id")
		return
	}

	// 1) Check if answering is allowed by engine
	state, round, ok := h.engine.CanAnswer(c.SessionID, questionID)
	if !ok || round == nil {
		h.sendError(c, "answer window closed")
		return
	}

	// 2) Ensure selected answer is one of the 4 options for this round
	isOption := false
	for _, id := range round.AnswerIDs {
		if id == answerID {
			isOption = true
			break
		}
	}
	if !isOption {
		h.sendError(c, "invalid answer option")
		return
	}

	// 3) Determine correctness
	isCorrect, err := h.repo.IsAnswerCorrect(ctx, questionID, answerID)
	if err != nil {
		h.sendError(c, "db error")
		return
	}

	// 4) Compute time taken in ms (optional; simple: from now to deadline)
	now := time.Now()
	remaining := round.Deadline.Sub(now)
	timeTakenMs := int(state.AnswerDuration.Milliseconds()) - int(remaining.Milliseconds())
	if timeTakenMs < 0 {
		timeTakenMs = 0
	}

	// 5) Insert submission and update score
	err = h.repo.InsertSubmissionAndUpdateScore(ctx, p.ID, questionID, answerID, isCorrect, timeTakenMs)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateSubmission) {
			h.sendError(c, "you already answered this question")
			return
		}
		h.sendError(c, "failed to store answer")
		return
	}

	_ = h.engine.BroadcastStateToParticipant(ctx, c.SessionID, p.ID)

	// 6) Optional: send an ACK just to this client
	ack := ws.ErrorPayload{Message: "answer received"}
	raw, _ := json.Marshal(ack)
	env := ws.Envelope{Type: ws.MsgError, Payload: raw} // or a dedicated MsgAnswerAck
	msg, _ := json.Marshal(env)
	c.Send <- msg
}

func (h *WSHandler) sendError(c *ws.Client, msg string) {
	payload := ws.ErrorPayload{Message: msg}
	raw, _ := json.Marshal(payload)
	env := ws.Envelope{Type: ws.MsgError, Payload: raw}
	data, _ := json.Marshal(env)
	c.Send <- data
}
