package handlers

import (
	"context"
	"kanji-quiz/pages"
	"kanji-quiz/pages/admin"
	"kanji-quiz/server/model"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
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

	if err := admin.SessionDetail(joinURL, quiz, session, participants).
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

func (h *AdminHandler) StartSession(c *gin.Context) {
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}

	duration, _ := strconv.Atoi(c.PostForm("duration_seconds"))
	if duration <= 0 {
		duration = 10 // sane default
	}

	session, err := h.repo.CreateSession(c.Request.Context(), quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	questionsWithAnswers, err := h.repo.GetQuestionsWithAnswers(c.Request.Context(), quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Build shuffled question list + 4 answers each
	rand.Shuffle(len(questionsWithAnswers), func(i, j int) {
		questionsWithAnswers[i], questionsWithAnswers[j] = questionsWithAnswers[j], questionsWithAnswers[i]
	})

	var sessQuestions []model.SessionQuestion
	for _, qwa := range questionsWithAnswers {
		if len(qwa.Answers) < 4 {
			continue // or handle differently
		}
		// pick 4 random answers; simple approach
		idx := rand.Perm(len(qwa.Answers))[:4]
		ansIDs := make([]uuid.UUID, 0, 4)
		for _, i := range idx {
			ansIDs = append(ansIDs, qwa.Answers[i].ID)
		}
		sessQuestions = append(sessQuestions, model.SessionQuestion{
			QuestionID: qwa.Question.ID,
			AnswerIDs:  ansIDs,
		})
	}

	h.live.Set(&model.SessionState{
		SessionID:        session.ID,
		QuizID:           quizID,
		CurrentIndex:     0,
		Questions:        sessQuestions,
		QuestionDuration: duration,
	})

	c.Redirect(http.StatusSeeOther, "/admin/sessions/"+session.ID.String())
}

func adminError(msg string) *templ.ComponentHandler {
	return templ.Handler(pages.AdminLogin(msg, ""))
}
