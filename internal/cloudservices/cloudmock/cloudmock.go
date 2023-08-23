package cloudmock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
)

type MockCloudService struct {
	types.CloudService
	flavors       map[string]*model.NodePool
	organizations map[string]bool
	apps          map[string]*model.App
}

var _ types.CloudService = &MockCloudService{}

type MockOption func(*MockCloudService)

func (s *MockCloudService) GetFlavorNodePool(flavorID string) (*model.NodePool, error) {
	nodePool, hasNodePool := s.flavors[flavorID]
	if !hasNodePool {
		return nil, errors.New("not such flavor")

	}
	return nodePool, nil
}

func WithFlavors(flavors map[string]*model.NodePool) MockOption {
	return func(s *MockCloudService) {
		s.flavors = flavors
	}
}

func WithOrganizations(organizations map[string]bool) MockOption {
	return func(s *MockCloudService) {
		s.organizations = organizations
	}
}

func WithClusterApps(apps map[string]*model.App) MockOption {
	return func(s *MockCloudService) {
		s.apps = apps
	}
}

func NewMockCloudService(mockCloudOptions ...MockOption) *MockCloudService {
	s := MockCloudService{}

	for _, mockCloudOption := range mockCloudOptions {
		mockCloudOption(&s)
	}

	return &s
}
