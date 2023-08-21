package clustermock

import (
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
