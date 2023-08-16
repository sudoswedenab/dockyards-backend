package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *handler) GetWhoami(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Error("error getting user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	user.Password = ""

	c.JSON(http.StatusOK, user)
}
