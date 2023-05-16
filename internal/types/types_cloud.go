package types

import "bitbucket.org/sudosweden/dockyards-backend/api/v1/model"

type CloudService interface {
	PrepareEnvironment(*model.Cluster, *model.NodePoolOptions) error
	CleanEnvironment() error
}
