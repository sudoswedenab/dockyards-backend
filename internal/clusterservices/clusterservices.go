package clusterservices

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
)

type ClusterService interface {
	CreateNodePool(*v1alpha2.Organization, *v1alpha1.Cluster, *v1alpha1.NodePool) (*v1alpha1.NodePoolStatus, error)
	GetAllClusters() (*[]v1.Cluster, error)
	GetSupportedVersions() ([]string, error)
	DeleteGarbage()
	GetCluster(string) (*v1alpha1.ClusterStatus, error)
	CollectMetrics() error
	GetNodePool(string) (*v1alpha1.NodePoolStatus, error)
	DeleteNodePool(*v1alpha2.Organization, string) error
	GetNodes(*v1alpha1.NodePool) (*v1alpha1.NodeList, error)
	GetNode(string) (*v1alpha1.NodeStatus, error)
}
