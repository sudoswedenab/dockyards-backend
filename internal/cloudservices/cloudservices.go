package cloudservices

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
)

type CloudService interface {
	PrepareEnvironment(*v1alpha2.Organization, *v1alpha1.Cluster, *v1alpha1.NodePool) (*CloudConfig, error)
	CleanEnvironment(*v1alpha2.Organization, *CloudConfig) error
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
