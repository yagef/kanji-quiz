package main

import (
	"context"
	"kanji-quiz/server/handlers"
	"kanji-quiz/server/repository"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf(err.Error())
		return
	}

	quizRepo := repository.New(db)
	ah := handlers.NewAdmin(quizRepo)
	admin := r.Group("/admin", handlers.AdminAuthMiddleware)
	{
		admin.GET("/dashboard", ah.ListQuizzes)
		admin.POST("/quizzes", ah.CreateQuiz)
		admin.GET("/quizzes/:quizID", ah.QuizDetail)
		admin.POST("/quizzes/:quizID/questions", ah.CreateQuestion)
		admin.POST("/quizzes/:quizID/sessions", ah.CreateSession)
		admin.GET("/quizzes/:quizID/questions/:questionID", ah.QuestionDetail)
		admin.POST("/quizzes/:quizID/questions/:questionID/answers", ah.AddAnswer)
		admin.POST("/quizzes/:quizID/questions/:questionID/correct", ah.SetCorrectAnswer)
		admin.GET("/sessions/:sessionID", ah.SessionDetail)
		admin.POST("/sessions/:sessionID/end", ah.EndSession)
		admin.GET("/qr", ah.QR)
		//deletion
		admin.POST("/quizzes/:quizID/delete", ah.DeleteQuiz)
		admin.POST("/quizzes/:quizID/questions/:questionID/delete", ah.DeleteQuestion)
		admin.POST("/quizzes/:quizID/questions/:questionID/answers/:answerID/delete", ah.DeleteAnswer)
		admin.POST("/quizzes/:quizID/sessions/:sessionID/delete", ah.DeleteSession)
	}

	uh := handlers.NewUser(quizRepo)
	user := r.Group("/user", handlers.UserAuthMiddleware)
	{
		user.GET("/sessions/:sessionID", uh.JoinSession)
		user.GET("/participants/play", uh.ParticipantPage)
	}
	r.NoRoute(func(c *gin.Context) {
		handlers.Handle404().ServeHTTP(c.Writer, c.Request)
	})
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf(err.Error())
		return
	}
}
