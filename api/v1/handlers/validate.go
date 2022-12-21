package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Validate(c *gin.Context) {
	r := Response(c)

	println("AUTH hit")

	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "user logged in",
		"status":     "success",
		"data":       gin.H{"user": r},
	})
}
