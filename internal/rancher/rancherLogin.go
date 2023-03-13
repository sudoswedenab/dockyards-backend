package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/backend/api/v1/model"
)

func (r *Rancher) RancherLogin(user model.User) (string, error) {
	NewRanchPWd, err := r.changeRancherPWD(user)
	if err != nil {
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
