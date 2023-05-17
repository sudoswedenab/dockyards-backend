package types

import "bitbucket.org/sudosweden/dockyards-backend/api/v1/model"

type CloudService interface {
	PrepareEnvironment(*model.Cluster, *model.NodePoolOptions) (*CloudConfig, error)
	CleanEnvironment(*CloudConfig) error
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
}
