package clustermock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
)

type MockClusterService struct {
	types.ClusterService
	clusters map[string]model.Cluster
}

type MockOption func(*MockClusterService)

func (s *MockClusterService) GetAllClusters() (*[]model.Cluster, error) {
	clusters := []model.Cluster{}

	for _, cluster := range s.clusters {
		clusters = append(clusters, cluster)
	}

	return &clusters, nil
}

func (s *MockClusterService) CreateCluster(organization *model.Organization, clusterOptions *model.ClusterOptions) (*model.Cluster, error) {
	_, hasCluster := s.clusters[clusterOptions.Name]
	if hasCluster {
		return nil, errors.New("cluster name in-use")

	}

	cluster := model.Cluster{
		ID:   "cluster-123",
		Name: clusterOptions.Name,
	}

	return &cluster, nil
}

func (s *MockClusterService) CreateNodePool(organization *model.Organization, cluster *model.Cluster, nodePoolOptions *model.NodePoolOptions) (*model.NodePool, error) {
	nodePool := model.NodePool{
		Name: nodePoolOptions.Name,
	}

	return &nodePool, nil
}

func (s *MockClusterService) DeleteCluster(organization *model.Organization, cluster *model.Cluster) error {
	for _, c := range s.clusters {
		if c.Organization == organization.Name && c.Name == cluster.Name {
			return nil
		}
	}

	return errors.New("no such cluster")

}

func WithClusters(clusters map[string]model.Cluster) MockOption {
	return func(s *MockClusterService) {
		s.clusters = clusters
	}
}

func NewMockClusterService(mockOptions ...MockOption) *MockClusterService {
	s := MockClusterService{}

	for _, mockOption := range mockOptions {
		mockOption(&s)
	}

	return &s
}

var _ types.ClusterService = &MockClusterService{}
