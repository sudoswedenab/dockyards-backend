package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Validate(c *gin.Context) {
	println("AUTH hit")

	c.JSON(http.StatusOK, gin.H{
		"hey": "user logged in",
	})

	// Get out info about user:
	user, _ := c.Get("user")

	c.JSON(http.StatusOK, gin.H{
		"message": user,
	})
}
