package handlers

import (
	"kanji-quiz/pages"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func UserLoginHandler(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodPost:
		userPostHandler(c.Writer, c.Request)
	case http.MethodGet:
		userGetHandler(c.Writer, c.Request)
	default:
		HandleError(http.StatusMethodNotAllowed, "Method are not allowed", "").ServeHTTP(c.Writer, c.Request)
	}
}

func userGetHandler(w http.ResponseWriter, r *http.Request) {
	returnURL := r.URL.Query().Get("next")
	_ = pages.UserLogin("", returnURL).Render(r.Context(), w)
}

func userPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		userError("Invalid form data").ServeHTTP(w, r)
		return
	}

	login := strings.TrimSpace(r.FormValue("login"))

	if login == "" {
		w.WriteHeader(http.StatusUnauthorized)
		userError("Login are required").ServeHTTP(w, r)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		logout(w, r)
		HandleErr(http.StatusInternalServerError, "Session error", err).ServeHTTP(w, r)
		return
	}

	session.Values["user_id"] = login
	session.Values["authenticated"] = true
	session.Values["is_admin"] = false

	if err := session.Save(r, w); err != nil {
		HandleErr(http.StatusInternalServerError, "Failed to save session", err).ServeHTTP(w, r)
		return
	}

	nextURL := safeReturnURL(r.FormValue("returnURL"))
	if nextURL == "" {
		nextURL = "/user/history"
	}
	http.Redirect(w, r, nextURL, http.StatusSeeOther)
}

func userError(msg string) *templ.ComponentHandler {
	return templ.Handler(pages.UserLogin(msg, ""))
}
