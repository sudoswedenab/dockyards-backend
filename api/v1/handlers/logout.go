package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Logout godoc
//
//	@Summary		Logout from app
//	@Tags			Login
//	@Produce		text/plain
//	@Success		200
//	@Router			/logout [post]
func Logout(c *gin.Context) {
	c.SetCookie("AccessToken", "", -1, "", "", false, true)
	c.SetCookie("RefreshToken", "", -1, "", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"status": "logged out",
	})
}
