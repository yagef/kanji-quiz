package handlers

import (
	"kanji-quiz/pages"
	"net/http"
	"os"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
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
		nextURL = "/dashboard"
	}
	http.Redirect(w, r, nextURL, http.StatusSeeOther)
}

func adminError(msg string) *templ.ComponentHandler {
	return templ.Handler(pages.AdminLogin(msg, ""))
}
