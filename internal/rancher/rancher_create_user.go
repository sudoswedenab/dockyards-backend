package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type RancherUserResponse struct {
	Id string `json:"id"`
}

func (r *Rancher) RancherCreateUser(user model.RancherUser) (string, error) {
	newUser := managementv3.User{
		Name: user.Name,
	}
	createdUser, err := r.ManagementClient.User.Create(&newUser)
	if err != nil {
		return "", err
	}

	err = r.BindRole(createdUser.ID, "dockyard-role")
	if err != nil {
		return "", err
	}

	return createdUser.ID, nil
}
