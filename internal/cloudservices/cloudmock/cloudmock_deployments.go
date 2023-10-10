package cloudmock

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
)

func (s *MockCloudService) GetClusterDeployments(organization *v1alpha1.Organization, cluster *v1alpha1.Cluster, nodePoolList *v1alpha1.NodePoolList) (*[]v1.Deployment, error) {
	deployments := []v1.Deployment{}
	for key := range s.deployments {
		deployment := *s.deployments[key]
		deployment.ClusterId = string(cluster.UID)

		deployments = append(deployments, deployment)
	}

	return &deployments, nil
}
