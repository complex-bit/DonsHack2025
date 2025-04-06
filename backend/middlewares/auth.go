package middlewares

import (
	"app/models"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionID := session.Get("userID")
		if sessionID == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		userId := sessionID.(uint)
		user := models.UserFromId(userId)
		if user.ID == 0 {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		c.Set("userID", user.ID)

		c.Next()
	}
}
