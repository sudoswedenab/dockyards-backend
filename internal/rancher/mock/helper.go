package mock

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal/types"
)

type MockRancherHelper struct {
	MockCreateCluster  func(*model.ClusterOptions) (*model.Cluster, error)
	MockCreateNodePool func(*model.Cluster, *model.NodePoolOptions) (*model.NodePool, error)
	MockGetAllClusters func() (*[]model.Cluster, error)
	MockDeleteCluster  func(string) error
	MockGetKubeConfig  func(*model.Cluster) (string, error)
}

func (h *MockRancherHelper) CreateCluster(c *model.ClusterOptions) (*model.Cluster, error) {
	return h.MockCreateCluster(c)
}

func (h *MockRancherHelper) CreateNodePool(c *model.Cluster, o *model.NodePoolOptions) (*model.NodePool, error) {
	return h.MockCreateNodePool(c, o)
}

func (h *MockRancherHelper) GetAllClusters() (*[]model.Cluster, error) {
	return h.MockGetAllClusters()
}

func (h *MockRancherHelper) DeleteCluster(clusterId string) error {
	return h.MockDeleteCluster(clusterId)
}

func (h *MockRancherHelper) GetSupportedVersions() []string {
	return []string{}
}

func (h *MockRancherHelper) GetKubeConfig(c *model.Cluster) (string, error) {
	return h.MockGetKubeConfig(c)
}

func (h *MockRancherHelper) DeleteGarbage() {
	return
}

var _ types.ClusterService = &MockRancherHelper{}
