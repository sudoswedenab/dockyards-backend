package types

import "bitbucket.org/sudosweden/dockyards-backend/api/v1/model"

type CloudService interface {
	PrepareEnvironment(*model.Organization, *model.Cluster, *model.NodePoolOptions) (*CloudConfig, error)
	CleanEnvironment(*model.Organization, *CloudConfig) error
	CreateOrganization(*model.Organization) (string, error)
	GetOrganization(*model.Organization) (string, error)
	GetClusterApps(*model.Organization, *model.Cluster) (*[]model.App, error)
	DeleteGarbage()
	GetFlavorNodePool(string) (*model.NodePool, error)
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
