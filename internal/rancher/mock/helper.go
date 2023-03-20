package mock

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal/rancher"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockRancherHelper struct {
	MockRancherCreateUser     func(model.RancherUser) (string, error)
	MockRancherCreateCluster  func(string, string, string, string) (managementv3.Cluster, error)
	MockRancherCreateNodePool func(string, string) (managementv3.NodePool, error)
	MockRancherLogin          func(model.User) (string, error)
	MockCreateClusterRole     func() error
}

func (h *MockRancherHelper) RancherCreateUser(user model.RancherUser) (string, error) {
	return h.MockRancherCreateUser(user)
}

func (h *MockRancherHelper) RancherCreateCluster(dockerRootDir, name, ctrId, ctId string) (managementv3.Cluster, error) {
	return h.MockRancherCreateCluster(dockerRootDir, name, ctrId, ctId)
}

func (h *MockRancherHelper) RancherCreateNodePool(id, name string) (managementv3.NodePool, error) {
	return h.MockRancherCreateNodePool(id, name)
}

func (h *MockRancherHelper) RancherLogin(user model.User) (string, error) {
	return h.MockRancherLogin(user)
}

func (h *MockRancherHelper) CreateClusterRole() error {
	return h.MockCreateClusterRole()
}

var _ rancher.RancherService = &MockRancherHelper{}
