package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/backend/internal"

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
	c.SetCookie(internal.AccessTokenName, "", -1, "", "", false, true)
	c.SetCookie(internal.RefreshTokenName, "", -1, "", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"status": "logged out",
	})
}
