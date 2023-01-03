package handlers

// Unnecessary/unused code?
import (
	"Backend/api/v1/handlers/user"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Validate godoc
//
//	@Summary		Validate a user
//	@Tags			Validate
//	@Accept			application/json
//	@Produce		application/json
//	@Param			request	body	model.User	true "User model"
//	@Success		200	{object}	model.User
//	@Router			/admin/auth [get]
func Validate(c *gin.Context) {
	r := user.Response(c)

	fmt.Println("AUTH hit")

	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "user logged in",
		"status":     "success",
		"data":       gin.H{"user": r},
	})
}
