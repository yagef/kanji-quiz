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
		public.POST("/login", handlers.UserLoginHandler)
		public.GET("/login", handlers.UserLoginHandler)
		public.POST("/admin", handlers.AdminLoginHandler)
		public.GET("/admin", handlers.AdminLoginHandler)
		public.GET("/logout", func(c *gin.Context) {
			handlers.LogoutHandler(c.Writer, c.Request)
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
