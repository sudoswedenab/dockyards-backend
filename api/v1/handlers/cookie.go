package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// ReadCookie godoc
//
//	@Summary		ReadCookie to app
//	@Tags			Cookie
//	@Accept			application/json
//	@Produce		text/plain
//	@Success		200
//	@Failure		400
//	@Router			/readcookie [get]
func ReadCookie(c *gin.Context) {
	kakor := c.Request.Cookies()
	fmt.Printf("dump cookies: %s\n", kakor)

	c.JSON(http.StatusOK, gin.H{
		"Read": "Success",
	})
}
