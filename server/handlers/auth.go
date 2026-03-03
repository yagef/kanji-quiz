package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(c *gin.Context) {
	r := c.Request
	w := c.Writer
	session, _ := store.Get(r, "session-name")

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {

		requestedURL := r.URL.RequestURI()
		loginURL := "/login?next=" + url.QueryEscape(requestedURL)

		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		c.Abort()
		return
	}
	c.Next()
}
