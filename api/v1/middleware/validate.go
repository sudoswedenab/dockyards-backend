package middleware

import (
	"Backend/api/v1/handlers/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Validate godoc
//
//	@Summary		Validate a user
//	@Tags			Login
//	@Accept			application/json
//	@Produce		application/json
//	@Param			request	body	model.User	true "User model"
//	@Success		200	{object}	model.User
//	@Router			/admin/auth [get]
func Validate(c *gin.Context) {
	r := user.Response(c)

	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "user logged in",
		"status":     "success",
		"data":       gin.H{"user": r},
	})
}
