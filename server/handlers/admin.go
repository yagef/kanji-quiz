package handlers

import (
	"context"
	"kanji-quiz/pages"
	"kanji-quiz/pages/admin"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

var adminPassword = os.Getenv("ADMIN_PASS")

func AdminLoginHandler(c *gin.Context) {
	r := c.Request
	w := c.Writer
	switch r.Method {
	case http.MethodPost:
		adminPostHandler(w, r)
	case http.MethodGet:
		adminGetHandler(w, r)
	default:
		HandleError(http.StatusMethodNotAllowed, "Method not allowed", "").ServeHTTP(w, r)
	}
}

func adminGetHandler(w http.ResponseWriter, r *http.Request) {
	returnURL := r.URL.Query().Get("next")
	_ = pages.AdminLogin("", returnURL).Render(r.Context(), w)
}

func adminPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		adminError("Invalid form data").ServeHTTP(w, r)
		return
	}

	password := strings.TrimSpace(r.FormValue("password"))

	if password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		adminError("Password are required").ServeHTTP(w, r)
		return
	}

	if password != adminPassword {
		w.WriteHeader(http.StatusUnauthorized)
		adminError("Incorrect password").ServeHTTP(w, r)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}

	session.Values["user_id"] = "admin"
	session.Values["authenticated"] = true
	session.Values["is_admin"] = true

	if err := session.Save(r, w); err != nil {
		HandleErr(http.StatusInternalServerError, "Failed to save session", err).ServeHTTP(w, r)
		return
	}

	nextURL := safeReturnURL(r.FormValue("returnURL"))
	if nextURL == "" {
		nextURL = "/admin/dashboard"
	}
	http.Redirect(w, r, nextURL, http.StatusSeeOther)
}

func (h *AdminHandler) DeleteQuiz(c *gin.Context) {
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}

	if err := h.repo.DeleteQuiz(c.Request.Context(), quizID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	// Go back to quiz list
	c.Redirect(http.StatusSeeOther, "/admin/dashboard")
}

func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}

	questionID, ok := mustUUID(c, c.Param("questionID"))
	if !ok {
		return
	}

	if err := h.repo.DeleteQuestion(c.Request.Context(), quizID, questionID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	// Go back to quiz detail
	c.Redirect(http.StatusSeeOther, "/admin/quizzes/"+quizID.String())
}

func (h *AdminHandler) DeleteAnswer(c *gin.Context) {
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}

	questionID, ok := mustUUID(c, c.Param("questionID"))
	if !ok {
		return
	}

	answerID, ok := mustUUID(c, c.Param("answerID"))
	if !ok {
		return
	}

	if err := h.repo.DeleteAnswer(c.Request.Context(), questionID, answerID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	// Go back to question detail
	c.Redirect(http.StatusSeeOther, "/admin/quizzes/"+quizID.String()+"/questions/"+questionID.String())
}

func (h *AdminHandler) QR(c *gin.Context) {
	sessionID := c.Query("session")
	if sessionID == "" {
		c.String(http.StatusBadRequest, "missing session")
		return
	}

	baseURL := os.Getenv("SERVER_BASE_URL")
	if baseURL == "" {
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		baseURL = scheme + "://" + c.Request.Host
	}

	content := baseURL + "/user/sessions/" + sessionID

	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate QR")
		return
	}

	c.Header("Content-Type", "image/png")
	_, _ = c.Writer.Write(png)
}

// SessionDetail Handler
func (h *AdminHandler) SessionDetail(c *gin.Context) {
	sessionID, ok := mustUUID(c, c.Param("sessionID"))
	if !ok {
		return
	}

	session, err := h.repo.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	quiz, err := h.repo.GetQuiz(c.Request.Context(), session.QuizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	participants, err := h.repo.ListParticipants(c.Request.Context(), sessionID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	baseURL := os.Getenv("SERVER_BASE_URL")
	if baseURL == "" {
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		baseURL = scheme + "://" + c.Request.Host
	}

	joinURL := baseURL + "/user/sessions/" + sessionID.String()

	phase := h.engine.GetPhase(sessionID)
	if err := admin.SessionDetail(joinURL, quiz, session, participants, phase).
		Render(context.Background(), c.Writer); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
}

// EndSession Handler
func (h *AdminHandler) EndSession(c *gin.Context) {
	sessionID, ok := mustUUID(c, c.Param("sessionID"))
	if !ok {
		return
	}

	if err := h.repo.EndSession(c.Request.Context(), sessionID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/sessions/"+sessionID.String())
}

func (h *AdminHandler) DeleteSession(c *gin.Context) {
	id, ok := mustUUID(c, c.Param("sessionID"))
	if !ok {
		return
	}

	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}

	if err := h.repo.DeleteSession(c.Request.Context(), id); err != nil {
		// You can distinguish not-found if you want
		c.Status(http.StatusInternalServerError)
		return
	}

	h.live.Delete(id)
	c.Redirect(http.StatusSeeOther, "/admin/quizzes/"+quizID.String())
}

// InitQuiz When admin sets time and presses Start in /admin/sessions/:sessionID
func (h *AdminHandler) InitQuiz(c *gin.Context) {
	sessionID, ok := mustUUID(c, c.Param("sessionID"))
	if !ok {
		return
	}

	// Parse form values with safe defaults
	answerSeconds, err := strconv.Atoi(c.PostForm("answer_seconds"))
	if err != nil || answerSeconds < 5 {
		answerSeconds = 15 // sane default
	}
	countdownSeconds, err := strconv.Atoi(c.PostForm("countdown_seconds"))
	if err != nil || countdownSeconds < 3 {
		countdownSeconds = 5
	}

	ctx := c.Request.Context()
	session, err := h.repo.GetSession(ctx, sessionID)
	if err != nil {
		c.String(http.StatusNotFound, "session not found")
		return
	}

	if err := h.engine.InitSession(ctx, sessionID, session.QuizID, answerSeconds, countdownSeconds); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.engine.StartQuiz(sessionID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.repo.ClearSessionAnswers(ctx, sessionID); err != nil {
		c.String(http.StatusInternalServerError, "failed to reset session: "+err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/sessions/"+sessionID.String())
}

func (h *AdminHandler) NextQuestion(c *gin.Context) {
	sessionID, ok := mustUUID(c, c.Param("sessionID"))
	if !ok {
		return
	}

	if err := h.engine.NextQuestion(sessionID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/sessions/"+sessionID.String())
}

func adminError(msg string) *templ.ComponentHandler {
	return templ.Handler(pages.AdminLogin(msg, ""))
}
