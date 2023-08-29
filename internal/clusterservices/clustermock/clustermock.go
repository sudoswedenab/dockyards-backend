package clustermock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
)

type MockClusterService struct {
	types.ClusterService
	clusters map[string]v1.Cluster
}

type MockOption func(*MockClusterService)

func (s *MockClusterService) GetAllClusters() (*[]v1.Cluster, error) {
	clusters := []v1.Cluster{}

	for _, cluster := range s.clusters {
		clusters = append(clusters, cluster)
	}

	return &clusters, nil
}

func (s *MockClusterService) CreateCluster(organization *v1.Organization, clusterOptions *v1.ClusterOptions) (*v1.Cluster, error) {
	_, hasCluster := s.clusters[clusterOptions.Name]
	if hasCluster {
		return nil, errors.New("cluster name in-use")

	}

	cluster := v1.Cluster{
		ID:   "cluster-123",
		Name: clusterOptions.Name,
	}

	return &cluster, nil
}

func (s *MockClusterService) CreateNodePool(organization *v1.Organization, cluster *v1.Cluster, nodePoolOptions *v1.NodePoolOptions) (*v1.NodePool, error) {
	nodePool := v1.NodePool{
		Name: nodePoolOptions.Name,
	}

	return &nodePool, nil
}

func (s *MockClusterService) DeleteCluster(organization *v1.Organization, cluster *v1.Cluster) error {
	for _, c := range s.clusters {
		if c.Organization == organization.Name && c.Name == cluster.Name {
			return nil
		}
	}

	return errors.New("no such cluster")

}

func (s *MockClusterService) GetCluster(clusterID string) (*v1.Cluster, error) {
	cluster, hasCluster := s.clusters[clusterID]
	if !hasCluster {
		return nil, errors.New("no such cluster")
	}

	return &cluster, nil
}

func (s *MockClusterService) CollectMetrics() error {
	return nil
}

func WithClusters(clusters map[string]v1.Cluster) MockOption {
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
