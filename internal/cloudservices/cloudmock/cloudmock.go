package cloudmock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
)

type MockCloudService struct {
	cloudservices.CloudService
	flavors       map[string]*v1.NodePool
	organizations map[string]bool
	deployments   map[string]*v1.Deployment
}

var _ cloudservices.CloudService = &MockCloudService{}

type MockOption func(*MockCloudService)

func (s *MockCloudService) GetFlavorNodePool(flavorID string) (*v1.NodePool, error) {
	nodePool, hasNodePool := s.flavors[flavorID]
	if !hasNodePool {
		return nil, errors.New("not such flavor")

	}
	return nodePool, nil
}

func (s *MockCloudService) DeleteGarbage() {
	return
}

func WithFlavors(flavors map[string]*v1.NodePool) MockOption {
	return func(s *MockCloudService) {
		s.flavors = flavors
	}
}

func WithOrganizations(organizations map[string]bool) MockOption {
	return func(s *MockCloudService) {
		s.organizations = organizations
	}
}

func WithClusterDeployments(deployments map[string]*v1.Deployment) MockOption {
	return func(s *MockCloudService) {
		s.deployments = deployments
	}
}

func NewMockCloudService(mockCloudOptions ...MockOption) *MockCloudService {
	s := MockCloudService{}

	for _, mockCloudOption := range mockCloudOptions {
		mockCloudOption(&s)
	}

	return &s
}
