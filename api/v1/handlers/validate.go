package handlers

import (
	"Backend/api/v1/handlers/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Validate godoc
//
//	@Summary		Validate user
//	@Description	validate a user
//	@Tags			Validate
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Account ID"
//	@Success		200	{object}	model.User
//	@Router			/auth [get]
func Validate(c *gin.Context) {
	r := user.Response(c)

	println("AUTH hit")

	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "user logged in",
		"status":     "success",
		"data":       gin.H{"user": r},
	})
}
