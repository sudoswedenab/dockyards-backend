package mock

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
)

type MockRancherHelper struct {
	MockCreateCluster  func(*model.Organization, *model.ClusterOptions) (*model.Cluster, error)
	MockCreateNodePool func(*model.Organization, *model.Cluster, *model.NodePoolOptions) (*model.NodePool, error)
	MockGetAllClusters func() (*[]model.Cluster, error)
	MockDeleteCluster  func(*model.Organization, *model.Cluster) error
	MockGetKubeConfig  func(*model.Cluster) (string, error)
	MockGetCluster     func(string) (*model.Cluster, error)
}

func (h *MockRancherHelper) CreateCluster(o *model.Organization, c *model.ClusterOptions) (*model.Cluster, error) {
	return h.MockCreateCluster(o, c)
}

func (h *MockRancherHelper) CreateNodePool(org *model.Organization, c *model.Cluster, o *model.NodePoolOptions) (*model.NodePool, error) {
	return h.MockCreateNodePool(org, c, o)
}

func (h *MockRancherHelper) GetAllClusters() (*[]model.Cluster, error) {
	return h.MockGetAllClusters()
}

func (h *MockRancherHelper) DeleteCluster(o *model.Organization, c *model.Cluster) error {
	return h.MockDeleteCluster(o, c)
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

func (h *MockRancherHelper) GetCluster(s string) (*model.Cluster, error) {
	return h.MockGetCluster(s)
}

var _ types.ClusterService = &MockRancherHelper{}
