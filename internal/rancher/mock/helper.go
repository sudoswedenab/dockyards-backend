package mock

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal/rancher"
)

type MockRancherHelper struct {
	MockRancherCreateUser func(model.RancherUser) (string, error)
	MockRancherLogin      func(model.User) (string, error)
}

func (h *MockRancherHelper) RancherCreateUser(user model.RancherUser) (string, error) {
	return h.MockRancherCreateUser(user)
}

func (h *MockRancherHelper) RancherLogin(user model.User) (string, error) {
	return h.MockRancherLogin(user)
}

var _ rancher.RancherService = &MockRancherHelper{}
