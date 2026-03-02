package handlers

import (
	"kanji-quiz/pages"
	"net/http"

	"github.com/a-h/templ"
)

func Handle404() *templ.ComponentHandler {
	return HandleError(http.StatusNotFound,
		"Page not found",
		"The page you are looking for does not exist, has been moved, or the URL is incorrect.")
}

func HandleLogin(msg string) *templ.ComponentHandler {
	return templ.Handler(pages.Login(msg))
}

func HandleError(code int, msg string, description string) *templ.ComponentHandler {
	return templ.Handler(pages.Error(code,
		msg,
		description))
}
