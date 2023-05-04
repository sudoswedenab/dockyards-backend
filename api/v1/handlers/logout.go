package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *handler) Logout(c *gin.Context) {
	c.SetCookie(h.accessTokenName, "", -1, "", "", false, true)
	c.SetCookie(h.refreshTokenName, "", -1, "", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"status": "logged out",
	})
}
