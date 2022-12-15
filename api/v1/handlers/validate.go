package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Validate(c *gin.Context) {
	println("AUTH hit")

	user, _ := c.Get("user")

	c.JSON(http.StatusOK, gin.H{
		"message": user,
	})
}
