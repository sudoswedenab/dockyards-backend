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

	RancherBearerToken, RancherUserID, err := CreateRancherToken(model.RRtoken{Name: user.Email, Password: NewRanchPWd})
	if err != nil {
		return "", err
	}
	if RancherUserID != user.RancherID {
		err := errors.New("invalid email or password")
		return "", err
	}

	return RancherBearerToken, nil
}
