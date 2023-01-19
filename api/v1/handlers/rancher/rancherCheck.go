package rancher

import (
	"Backend/api/v1/model"
	"errors"
	"fmt"
)

func RancherCheck(user model.User) (string, error) {

	NewRanchPWd, err := ChangeRancherPWD(user)
	if err != nil {
		return "", err
	}

	RancherBearerToken, RancherUserID := CreateRancherToken(model.RRtoken{Name: user.Name, Password: NewRanchPWd})
	fmt.Println(RancherBearerToken)

	if RancherUserID != user.RancherID {

		// c.JSON(http.StatusBadRequest, gin.H{
		// 	"error": "Invalid email",
		err := errors.New("invalid email or password")
		return "", err
	}
	// fmt.Println("DBUSER", user.RancherID)
	// fmt.Println(RancherUserID)

	return RancherBearerToken, nil
}
