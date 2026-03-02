package main

import (
	"kanji-quiz/server/handlers"
	"log"
	"net/http"

	"kanji-quiz/pages"

	"github.com/a-h/templ"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /login", templ.Handler(pages.Login("")))
	mux.HandleFunc("POST /login", handlers.Login)
	mux.Handle("/", handlers.Handle404())

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf(err.Error())
		return
	}
}
