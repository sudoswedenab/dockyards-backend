package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ReadCookie(c *gin.Context) {
	kakor := c.Request.Cookies()
	fmt.Printf("dump cookies: %s\n", kakor)

	c.JSON(http.StatusOK, gin.H{
		"Read": "Success",
	})
}
