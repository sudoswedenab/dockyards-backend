package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/backend/api/v1/model"
)

func (r *Rancher) RancherLogin(user model.User) (string, error) {
	rancherUser, err := r.ManagementClient.User.ByID(user.RancherID)
	if err != nil {
		return "", err
	}

	NewRanchPWd, err := r.changeRancherPWD(*rancherUser)
	if err != nil {
		return "", err
	}

	RancherBearerToken, RancherUserID, err := r.createRancherToken(model.RRtoken{Name: user.Email, Password: NewRanchPWd})
	if err != nil {
		return "", err
	}
	if RancherUserID != user.RancherID {
		err := errors.New("invalid email or password")
		return "", err
	}

	return RancherBearerToken, nil
}
