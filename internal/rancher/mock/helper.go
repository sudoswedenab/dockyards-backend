package mock

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal/rancher"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockRancherHelper struct {
	MockRancherCreateUser     func(model.RancherUser) (string, error)
	MockRancherCreateCluster  func(model.ClusterOptions) (managementv3.Cluster, error)
	MockRancherCreateNodePool func(model.ClusterOptions, string) (managementv3.NodePool, error)
	MockRancherLogin          func(model.User) (string, error)
	MockCreateClusterRole     func() error
}

func (h *MockRancherHelper) RancherCreateUser(user model.RancherUser) (string, error) {
	return h.MockRancherCreateUser(user)
}

func (h *MockRancherHelper) RancherCreateCluster(c model.ClusterOptions) (managementv3.Cluster, error) {
	return h.MockRancherCreateCluster(c)
}

func (h *MockRancherHelper) RancherCreateNodePool(c model.ClusterOptions, name string) (managementv3.NodePool, error) {
	return h.MockRancherCreateNodePool(c, name)
}

func (h *MockRancherHelper) RancherLogin(user model.User) (string, error) {
	return h.MockRancherLogin(user)
}

func (h *MockRancherHelper) CreateClusterRole() error {
	return h.MockCreateClusterRole()
}

var _ rancher.RancherService = &MockRancherHelper{}
