package handlers

import (
	"fmt"
	"kanji-quiz/pages/admin"
	"kanji-quiz/server/repository"
	"kanji-quiz/server/ws"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	repo   *repository.QuizRepo
	live   *ws.Manager
	engine *ws.Engine
}

func NewAdmin(repo *repository.QuizRepo, live *ws.Manager, engine *ws.Engine) *AdminHandler {
	return &AdminHandler{repo: repo, live: live, engine: engine}
}

func (h *AdminHandler) CreateQuiz(c *gin.Context) {
	title := c.PostForm("title")
	if title == "" {
		c.String(http.StatusBadRequest, "title required")
		return
	}
	quiz, err := h.repo.CreateQuiz(c.Request.Context(), title)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/quizzes/"+quiz.ID.String())
}

func (h *AdminHandler) ListQuizzes(c *gin.Context) {
	context := c.Request.Context()
	quizzes, err := h.repo.ListQuizzes(context)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	err = admin.Quizzes(quizzes).Render(context, c.Writer)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
}

func mustUUID(c *gin.Context, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return uuid.Nil, false
	}
	return id, true
}

func (h *AdminHandler) QuizDetail(c *gin.Context) {
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}

	context := c.Request.Context()
	quiz, err := h.repo.GetQuiz(context, quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	questions, err := h.repo.ListQuestions(context, quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	answerTypes, err := h.repo.ListAnswerTypes(context)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	sessions, err := h.repo.ListSessions(context, quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	err = admin.QuizDetail(quiz, questions, answerTypes, sessions).Render(context, c.Writer)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *AdminHandler) CreateSession(c *gin.Context) {
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}
	s, err := h.repo.CreateSession(c.Request.Context(), quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/sessions/"+s.ID.String())
}

func (h *AdminHandler) CreateQuestion(c *gin.Context) {
	r := c.Request
	w := c.Writer
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}
	kanji := r.FormValue("kanji")

	var typeID int
	_, err := fmt.Sscan(r.FormValue("type_id"), &typeID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	q, err := h.repo.CreateQuestion(r.Context(), quizID, typeID, kanji)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/quizzes/%s/questions/%s", quizID, q.ID), http.StatusSeeOther)
}

func (h *AdminHandler) QuestionDetail(c *gin.Context) {
	r := c.Request
	w := c.Writer
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}
	questionID, ok := mustUUID(c, c.Param("questionID"))
	if !ok {
		return
	}
	context := r.Context()
	quiz, err := h.repo.GetQuiz(context, quizID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	question, err := h.repo.GetQuestion(context, questionID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	answers, err := h.repo.ListAnswers(context, questionID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	err = admin.QuestionDetail(quiz, question, answers).Render(c.Request.Context(), w)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *AdminHandler) AddAnswer(c *gin.Context) {
	r := c.Request
	w := c.Writer
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}
	questionID, ok := mustUUID(c, c.Param("questionID"))
	if !ok {
		return
	}
	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	if _, err := h.repo.AddAnswer(r.Context(), questionID, text); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/quizzes/%s/questions/%s", quizID, questionID), http.StatusSeeOther)
}

func (h *AdminHandler) SetCorrectAnswer(c *gin.Context) {
	r := c.Request
	w := c.Writer
	quizID, ok := mustUUID(c, c.Param("quizID"))
	if !ok {
		return
	}
	questionID, ok := mustUUID(c, c.Param("questionID"))
	if !ok {
		return
	}
	answerID, ok := mustUUID(c, c.PostForm("answer_id"))
	if !ok {
		return
	}
	if err := h.repo.SetCorrectAnswer(r.Context(), questionID, answerID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/quizzes/%s/questions/%s", quizID, questionID), http.StatusSeeOther)
}
