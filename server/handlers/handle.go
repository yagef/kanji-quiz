package handlers

import (
	"encoding/base64"
	"kanji-quiz/pages"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func init() {
	authKeyStr := os.Getenv("SESSION_AUTH_KEY")
	encryptKeyStr := os.Getenv("SESSION_ENCRYPT_KEY")

	authKey, err := base64.StdEncoding.DecodeString(authKeyStr)
	if err != nil {
		log.Fatalf("failed to decode SESSION_AUTH_KEY: %v", err)
	}

	encryptKey, err := base64.StdEncoding.DecodeString(encryptKeyStr)
	if err != nil {
		log.Fatalf("failed to decode SESSION_ENCRYPT_KEY: %v", err)
	}

	store = sessions.NewCookieStore(authKey, encryptKey)
}

func Handle404() *templ.ComponentHandler {
	return HandleError(http.StatusNotFound,
		"Page not found",
		"The page you are looking for does not exist, has been moved, or the URL is incorrect.")
}

func HandleErr(code int, msg string, err error) *templ.ComponentHandler {
	log.Println(err.Error())
	return templ.Handler(pages.Error(code,
		msg,
		""))
}

func HandleError(code int, msg string, description string) *templ.ComponentHandler {
	return templ.Handler(pages.Error(code,
		msg,
		description))
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	logout(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1 // Delete the cookie
	_ = session.Save(r, w)
}

func safeReturnURL(u string) string {
	if strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//") {
		return u
	}
	return ""
}
