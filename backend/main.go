package main

import (
	"app/controllers"
	"app/middlewares"
	"app/models"
	"log"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	r := gin.Default()

	store := memstore.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("blogs", store))

	r.Use(gin.Logger())

	r.LoadHTMLGlob("templates/**/*")

	models.ConnectDatabase()
	models.DBMigrate()

	courses := r.Group("/courses", middlewares.AuthMiddleware())
	{
		courses.GET("/", controllers.CoursesIndex)
		courses.GET("/:id", controllers.CoursesShow)
	}
	r.GET("/users/signup", controllers.SignupPage)
	r.GET("/users/login", controllers.LoginPage)
	r.POST("/users/signup", controllers.Signup)
	r.POST("/users/login", controllers.Login)
	r.DELETE("/users/logout", controllers.Logout)

	log.Println("Server started!")
	r.Run() 
}
