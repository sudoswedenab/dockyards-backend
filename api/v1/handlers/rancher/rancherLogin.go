package rancher

import (
	"Backend/api/v1/model"
	"errors"
	"fmt"
)

func RancherLogin(user model.User) (string, error) {

	NewRanchPWd, err := ChangeRancherPWD(user)
	if err != nil {
		fmt.Println("new ranch pwd err check")
		return "", err
	}
	// fmt.Println("AVSTÄMD", user.Name)
	RancherBearerToken, RancherUserID, err := CreateRancherToken(model.RRtoken{Name: user.Email, Password: NewRanchPWd})
	if err != nil {
		return "", err
	}
	// fmt.Println("AV", user.Email)
	// fmt.Println("UserRancherNY", user)

	// fmt.Println(RancherBearerToken)

	// fmt.Println("Lola")
	// fmt.Println("Lola1", user.RancherID)
	// fmt.Println("Lola2", RancherUserID)
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
