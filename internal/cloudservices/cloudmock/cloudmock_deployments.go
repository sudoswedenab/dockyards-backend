package cloudmock

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
)

func (s *MockCloudService) GetClusterDeployments(organization *model.Organization, cluster *model.Cluster) (*[]model.Deployment, error) {
	deployments := []model.Deployment{}
	for key := range s.deployments {
		deployments = append(deployments, *s.deployments[key])
	}

	return &deployments, nil
}
