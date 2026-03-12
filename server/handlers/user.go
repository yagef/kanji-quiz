package handlers

import (
	"kanji-quiz/pages"
	"kanji-quiz/pages/user"
	"kanji-quiz/server/repository"
	"kanji-quiz/server/ws"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	repo *repository.QuizRepo
	live *ws.Manager
}

func NewUser(repo *repository.QuizRepo) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) JoinSession(c *gin.Context) {
	sessionID, ok := mustUUID(c, c.Param("sessionID"))
	if !ok {
		return
	}

	w := c.Writer
	r := c.Request
	userSession, err := store.Get(c.Request, "session-name")
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}

	name := userSession.Values["user_id"].(string)
	if name == "" {
		HandleError(http.StatusBadRequest, "Name is required", "").ServeHTTP(w, r)
		return
	}

	session, err := h.repo.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.String(http.StatusNotFound, "Session not found")
		return
	}
	if session.EndedAt != nil {
		c.String(http.StatusBadRequest, "Session already ended")
		return
	}

	u, err := h.repo.GetOrCreateUserByName(c.Request.Context(), name)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := h.repo.GetParticipantByUserAndSession(c.Request.Context(), u.ID, sessionID); err == nil {
		logout(w, r)
		err := pages.UserLogin("User with same name already a participant",
			"/user/sessions/"+sessionID.String()).Render(r.Context(), w)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}
		return
	}

	p, err := h.repo.CreateParticipant(c.Request.Context(), u.ID, sessionID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	userSession.Values["participant_id"] = p.ID.String()
	err = userSession.Save(r, w)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/user/participants/play")
}

func (h *UserHandler) ParticipantPage(c *gin.Context) {
	w := c.Writer
	r := c.Request
	userSession, err := store.Get(c.Request, "session-name")
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}

	id, ok := userSession.Values["participant_id"].(string)
	if !ok || id == "" {
		logout(w, r)
		HandleError(http.StatusInternalServerError, "Failed get participant ID", "").ServeHTTP(w, r)
		return
	}

	participantID, err := uuid.Parse(id)
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}

	// Load participant, join session & quiz
	p, err := h.repo.GetParticipant(c.Request.Context(), participantID)
	if err != nil {
		logout(w, r)
		c.String(http.StatusNotFound, "Participant not found")
		return
	}

	session, err := h.repo.GetSession(c.Request.Context(), p.SessionID)
	if err != nil {
		c.String(http.StatusNotFound, "Session not found")
		return
	}
	if session.EndedAt != nil {
		c.String(http.StatusBadRequest, "Quiz already ended")
		return
	}

	quiz, err := h.repo.GetQuiz(c.Request.Context(), session.QuizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	render(c, http.StatusOK, user.ParticipantPage(quiz, session, p))
}
