package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/handlers/user"

	"github.com/gin-gonic/gin"
)

func Validate(c *gin.Context) {
	r := user.Response(c)

	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "user logged in",
		"status":     "success",
		"data":       gin.H{"user": r},
	})
}
