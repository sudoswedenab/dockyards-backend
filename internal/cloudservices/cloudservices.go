package cloudservices

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
)

type CloudService interface {
	PrepareEnvironment(*v1alpha1.Organization, *v1.Cluster, *v1.NodePoolOptions) (*CloudConfig, error)
	CleanEnvironment(*v1alpha1.Organization, *CloudConfig) error
	GetClusterDeployments(*v1alpha1.Organization, *v1alpha1.Cluster, *v1alpha1.NodePoolList) (*[]v1.Deployment, error)
	DeleteGarbage()
	GetFlavorNodePool(string) (*v1.NodePool, error)
}

type CloudConfig struct {
	AuthURL                     string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	FlavorID                    string
	ImageID                     string
	KeypairName                 string
	NetID                       string
	PrivateKeyFile              string
	SecurityGroups              []string
}
