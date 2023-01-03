package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Logout godoc
//
//	@Summary		Logout from app
//	@Tags			Logout
//	@Produce		text/plain
//	@Success		200
//	@Router			/logout [post]
func Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "", "", false, true)
	c.SetCookie("refresh_token", "", -1, "", "", false, true)
	c.String(http.StatusOK, fmt.Sprintf("Logged out"))
}
