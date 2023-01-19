package rancher

import (
	"Backend/api/v1/model"
	"fmt"
)

func RancherCheck(user model.User) string {

	NewRanchPWd := ChangeRancherPWD(user)
	RancherBearerToken, RancherUserID := CreateRancherToken(model.RRtoken{Name: user.Name, Password: NewRanchPWd})
	fmt.Println(RancherBearerToken)

	if RancherUserID != user.RancherID {
		// c.JSON(http.StatusBadRequest, gin.H{
		// 	"error": "Invalid email",
		return ""
	}
	// fmt.Println("DBUSER", user.RancherID)
	// fmt.Println(RancherUserID)

	// c.JSON(http.StatusOK, gin.H{
	// 	"UserStatus": "user logged in",
	// })
	return RancherBearerToken
}
