package controllers

import (
	"app/models"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

func SignupPage(c *gin.Context) {
	c.HTML(http.StatusOK,
		"users/signup.tpl",
		gin.H{})
}

func LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK,
		"users/login.tpl",
		gin.H{})
}

type formData struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Canvas   string `form:"canvas"`
}


func Signup(c *gin.Context) {
	var data formData
	c.Bind(&data)

	if !models.CheckUserAvailability(data.Email) {
		c.Render(http.StatusBadRequest, render.Data{})
		return
	}

	user := models.UserCreate(data.Email, data.Password, data.Canvas)
	if user == nil || user.ID == 0 {
		c.Render(http.StatusBadRequest, render.Data{})
		return
	}
	session := sessions.Default(c)
	session.Set("userID", user.ID)
	session.Set("canvas", user.Canvas)
	session.Save()
	c.Redirect(http.StatusFound, "/courses")

}

func Login(c *gin.Context) {
	var data formData
	c.Bind(&data)

	// Match password
	user := models.UserMatchPassword(data.Email, data.Password)
	if user.ID == 0 {
		c.Render(http.StatusUnauthorized, render.Data{})
		return
	}
	// Set the session.
	session := sessions.Default(c)
	session.Set("userID", user.ID)
	session.Set("canvas", user.Canvas)
	session.Save()

	c.Redirect(http.StatusFound, "/canvas")
}

func Logout(c *gin.Context) {
	// Delete the session
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Status(http.StatusAccepted)
}
