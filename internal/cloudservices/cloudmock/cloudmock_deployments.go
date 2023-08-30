package cloudmock

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
)

func (s *MockCloudService) GetClusterDeployments(organization *v1.Organization, cluster *v1.Cluster) (*[]v1.Deployment, error) {
	deployments := []v1.Deployment{}
	for key := range s.deployments {
		deployment := *s.deployments[key]
		deployment.ClusterID = cluster.ID

		deployments = append(deployments, deployment)
	}

	return &deployments, nil
}
