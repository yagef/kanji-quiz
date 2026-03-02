package handlers

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

// Keys: 32-64 bytes auth key + optional 16/24/32 bytes encryption key
var store = sessions.NewCookieStore(
	[]byte(os.Getenv("SESSION_AUTH_KEY")),    // HMAC signing key
	[]byte(os.Getenv("SESSION_ENCRYPT_KEY")), // AES encryption key (optional)
)

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		HandleLogin("Invalid form data").ServeHTTP(w, r)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		HandleLogin("Login and password are required").ServeHTTP(w, r)
		return
	}

	if login != "admin" || password != "pswrd" {
		w.WriteHeader(http.StatusUnauthorized)
		HandleLogin("Incorrect login or password").ServeHTTP(w, r)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		HandleError(http.StatusMethodNotAllowed, "Method not allowed", "").ServeHTTP(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		HandleLogin("Invalid form data").ServeHTTP(w, r)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		w.WriteHeader(http.StatusUnauthorized)
		HandleLogin("Login and password are required").ServeHTTP(w, r)
		return
	}

	if login != "admin" || password != "pswrd" {
		w.WriteHeader(http.StatusUnauthorized)
		HandleLogin("Incorrect login or password").ServeHTTP(w, r)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		HandleError(http.StatusInternalServerError, "Session error", "").ServeHTTP(w, r)
		return
	}

	session.Values["user_id"] = login
	session.Values["authenticated"] = true

	if err := session.Save(r, w); err != nil {
		HandleError(http.StatusInternalServerError, "Failed to save session", "").ServeHTTP(w, r)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1 // Delete the cookie
	_ = session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
