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

	if !models.CheckUser(data.Email) {
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
	session := sessions.Default(c)
	var id int = 11

	//TODO : Look for existing Users in database and get the canvas/ID
	var uintID uint64 = uint64(id)
	session.Set("userID", uintID)
	session.Set("canvas", "abc123")
	session.Save()
	c.Redirect(http.StatusFound, "/courses")
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Status(http.StatusAccepted)
}
