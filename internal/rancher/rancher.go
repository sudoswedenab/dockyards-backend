package rancher

import "bitbucket.org/sudosweden/backend/api/v1/model"

type RancherService interface {
	RancherCreateUser(model.RancherUser) (string, error)
	RancherLogin(model.User) (string, error)
}

type Rancher struct {
	BearerToken string
	Url         string
}

var _ RancherService = &Rancher{}
