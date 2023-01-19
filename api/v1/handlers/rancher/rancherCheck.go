package rancher

import (
	"Backend/api/v1/model"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RancherCheck(c *gin.Context, user model.User) *gin.Context {

	NewRanchPWd := ChangeRancherPWD(c, user)
	RancherBearerToken, RancherUserID := CreateRancherToken(c, model.RRtoken{Name: user.Name, Password: NewRanchPWd})
	fmt.Println(RancherBearerToken)

	if RancherUserID != user.RancherID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email",
		})
	}
	fmt.Println("DBUSER", user.RancherID)
	fmt.Println(RancherUserID)

	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "user logged in",
	})
	return c
}
