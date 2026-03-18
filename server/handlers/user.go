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
		u, err := h.repo.GetOrCreateUserByName(c.Request.Context(), name)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		p, err := h.repo.GetParticipantByUserAndSession(c.Request.Context(), u.ID, sessionID)
		if err != nil {
			// User never participated in this session — send them home
			c.Redirect(http.StatusSeeOther, "/user/history")
			return
		}
		// User did participate — send them straight to their results
		c.Redirect(http.StatusSeeOther, "/user/participants/"+p.ID.String()+"/results")
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

// History shows all quiz sessions the logged-in user has taken.
func (h *UserHandler) History(c *gin.Context) {
	w, r := c.Writer, c.Request
	userSession, err := store.Get(r, "session-name")
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}
	name, _ := userSession.Values["user_id"].(string)
	if name == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	entries, err := h.repo.GetUserHistory(r.Context(), name)
	if err != nil {
		HandleErr(http.StatusInternalServerError, "Could not load history", err).ServeHTTP(w, r)
		return
	}
	render(c, http.StatusOK, user.HistoryPage(name, entries))
}

// SessionResult shows every answer the user gave in one past session.
func (h *UserHandler) SessionResult(c *gin.Context) {
	w, r := c.Writer, c.Request

	participantID, ok := mustUUID(c, c.Param("participantID"))
	if !ok {
		return
	}

	// Security: verify the participant belongs to the logged-in user
	userSession, err := store.Get(r, "session-name")
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}
	name, _ := userSession.Values["user_id"].(string)

	p, err := h.repo.GetParticipant(r.Context(), participantID)
	if err != nil {
		HandleError(http.StatusNotFound, "Participant not found", "").ServeHTTP(w, r)
		return
	}
	// make sure this participant actually belongs to the current user
	u, err := h.repo.GetOrCreateUserByName(r.Context(), name)
	if err != nil || u.ID != p.UserID {
		HandleError(http.StatusForbidden, "Access denied", "").ServeHTTP(w, r)
		return
	}

	quiz, err := h.repo.GetSessionQuiz(r.Context(), p.SessionID)
	if err != nil {
		HandleErr(http.StatusInternalServerError, "Could not load quiz", err).ServeHTTP(w, r)
		return
	}

	submissions, err := h.repo.GetParticipantSubmissions(r.Context(), participantID)
	if err != nil {
		HandleErr(http.StatusInternalServerError, "Could not load submissions", err).ServeHTTP(w, r)
		return
	}
	render(c, http.StatusOK, user.SessionResultPage(quiz, p, submissions))
}
