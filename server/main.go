package main

import (
	"log"
	"net/http"

	"kanji-quiz/pages"

	"github.com/a-h/templ"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /login", templ.Handler(pages.Login("")))
	mux.HandleFunc("POST /login", loginHandler)
	mux.Handle("/", templ.Handler(Error404()))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf(err.Error())
		return
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templ.Handler(pages.Login("Invalid form data")).ServeHTTP(w, r)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	if login != "admin" || password != "password" {
		w.WriteHeader(http.StatusUnauthorized)
		templ.Handler(pages.Login("Incorrect login or password")).ServeHTTP(w, r)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func Error404() templ.Component {
	return pages.Error(http.StatusNotFound,
		"Page not found",
		"The page you are looking for does not exist, has been moved, or the URL is incorrect.")
}
