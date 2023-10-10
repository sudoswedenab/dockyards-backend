package clusterservices

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

type ClusterService interface {
	CreateCluster(*v1alpha1.Organization, *v1.ClusterOptions) (*v1.Cluster, error)
	CreateNodePool(*v1alpha1.Organization, *v1.Cluster, *v1.NodePoolOptions) (*v1alpha1.NodePoolStatus, error)
	GetAllClusters() (*[]v1.Cluster, error)
	DeleteCluster(*v1alpha1.Organization, *v1.Cluster) error
	GetSupportedVersions() ([]string, error)
	DeleteGarbage()
	GetCluster(string) (*v1alpha1.ClusterStatus, error)
	CollectMetrics() error
	GetNodePool(string) (*v1alpha1.NodePoolStatus, error)
	GetKubeconfig(string, time.Duration) (*clientcmdv1.Config, error)
	DeleteNodePool(*v1alpha1.Organization, string) error
	GetNodes(*v1alpha1.NodePool) (*v1alpha1.NodeList, error)
}
