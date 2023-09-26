package cloudservices

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
)

type CloudService interface {
	PrepareEnvironment(*v1.Organization, *v1.Cluster, *v1.NodePoolOptions) (*CloudConfig, error)
	CleanEnvironment(*v1.Organization, *CloudConfig) error
	GetClusterDeployments(*v1.Organization, *v1.Cluster) (*[]v1.Deployment, error)
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
