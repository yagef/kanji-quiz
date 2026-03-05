package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func UserAuthMiddleware(c *gin.Context) {
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

func AdminAuthMiddleware(c *gin.Context) {
	r := c.Request
	w := c.Writer
	session, _ := store.Get(r, "session-name")

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {

		requestedURL := r.URL.RequestURI()
		loginURL := "/admin?next=" + url.QueryEscape(requestedURL)

		http.Redirect(w, r, loginURL, http.StatusSeeOther)
		c.Abort()
		return
	}
	if admin, ok := session.Values["is_admin"].(bool); !ok || !admin {

		HandleError(http.StatusUnauthorized, "Access Denied", "Page require admin permissions.").ServeHTTP(w, r)
		c.Abort()
		return
	}
	c.Next()
}
