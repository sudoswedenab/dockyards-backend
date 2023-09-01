package types

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
)

type ClusterService interface {
	CreateCluster(*v1.Organization, *v1.ClusterOptions) (*v1.Cluster, error)
	CreateNodePool(*v1.Organization, *v1.Cluster, *v1.NodePoolOptions) (*v1.NodePool, error)
	GetAllClusters() (*[]v1.Cluster, error)
	DeleteCluster(*v1.Organization, *v1.Cluster) error
	GetSupportedVersions() ([]string, error)
	DeleteGarbage()
	GetCluster(string) (*v1.Cluster, error)
	CollectMetrics() error
	GetNodePool(string) (*v1.NodePool, error)
	GetKubeconfig(string) (string, error)
}
