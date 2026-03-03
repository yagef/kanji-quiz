package main

import (
	"kanji-quiz/server/handlers"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	public := r.Group("/")
	{
		public.Any("/login", handlers.UserLoginHandler)
		public.Any("/logout", func(c *gin.Context) {
			handlers.LogoutHandler(c.Writer, c.Request)
		})
		public.Any("/admin", func(c *gin.Context) {
			handlers.AdminLoginHandler(c.Writer, c.Request)
		})
	}

	/*protected := r.Group("/", handlers.AuthMiddleware)
	{
	}*/
	r.NoRoute(func(c *gin.Context) {
		handlers.Handle404().ServeHTTP(c.Writer, c.Request)
	})
	err := r.Run(":8080")
	if err != nil {
		log.Fatalf(err.Error())
		return
	}
}
