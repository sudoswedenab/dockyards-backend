package types

import "bitbucket.org/sudosweden/backend/api/v1/model"

type ClusterService interface {
	CreateCluster(*model.Organization, *model.ClusterOptions) (*model.Cluster, error)
	CreateNodePool(*model.Cluster, *model.NodePoolOptions) (*model.NodePool, error)
	GetAllClusters() (*[]model.Cluster, error)
	DeleteCluster(*model.Cluster) error
	GetSupportedVersions() []string
	GetKubeConfig(*model.Cluster) (string, error)
	DeleteGarbage()
}
