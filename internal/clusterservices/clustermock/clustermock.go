package clustermock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

type MockClusterService struct {
	clusterservices.ClusterService
	clusters    map[string]v1.Cluster
	nodePools   map[string]v1.NodePool
	kubeconfigs map[string]clientcmdv1.Config
}

type MockOption func(*MockClusterService)

func (s *MockClusterService) GetAllClusters() (*[]v1.Cluster, error) {
	clusters := []v1.Cluster{}

	for _, cluster := range s.clusters {
		clusters = append(clusters, cluster)
	}

	return &clusters, nil
}

func (s *MockClusterService) CreateCluster(organization *v1alpha1.Organization, clusterOptions *v1.ClusterOptions) (*v1.Cluster, error) {
	_, hasCluster := s.clusters[clusterOptions.Name]
	if hasCluster {
		return nil, errors.New("cluster name in-use")

	}

	cluster := v1.Cluster{
		ID:   "cluster-123",
		Name: clusterOptions.Name,
	}

	return &cluster, nil
}

func (s *MockClusterService) CreateNodePool(organization *v1.Organization, cluster *v1.Cluster, nodePoolOptions *v1.NodePoolOptions) (*v1.NodePool, error) {
	nodePool := v1.NodePool{
		Name:                       nodePoolOptions.Name,
		Quantity:                   nodePoolOptions.Quantity,
		ControlPlane:               nodePoolOptions.ControlPlane,
		ControlPlaneComponentsOnly: nodePoolOptions.ControlPlaneComponentsOnly,
		Etcd:                       nodePoolOptions.Etcd,
		LoadBalancer:               nodePoolOptions.LoadBalancer,
	}

	if nodePoolOptions.RAMSizeMb != nil {
		nodePool.RAMSizeMb = *nodePoolOptions.RAMSizeMb
	}

	if nodePoolOptions.DiskSizeGb != nil {
		nodePool.DiskSizeGb = *nodePoolOptions.DiskSizeGb
	}

	if nodePoolOptions.CPUCount != nil {
		nodePool.CPUCount = *nodePoolOptions.CPUCount
	}

	return &nodePool, nil
}

func (s *MockClusterService) DeleteCluster(organization *v1.Organization, cluster *v1.Cluster) error {
	for _, c := range s.clusters {
		if c.Organization == organization.Name && c.Name == cluster.Name {
			return nil
		}
	}

	return errors.New("no such cluster")

}

func (s *MockClusterService) GetCluster(clusterID string) (*v1.Cluster, error) {
	cluster, hasCluster := s.clusters[clusterID]
	if !hasCluster {
		return nil, errors.New("no such cluster")
	}

	return &cluster, nil
}

func (s *MockClusterService) CollectMetrics() error {
	return nil
}

func (s *MockClusterService) DeleteGarbage() {
	return
}

func (s *MockClusterService) GetNodePool(nodePoolID string) (*v1.NodePool, error) {
	nodePool, hasNodePool := s.nodePools[nodePoolID]
	if !hasNodePool {
		return nil, errors.New("no such node pool")
	}

	return &nodePool, nil
}

func (s *MockClusterService) DeleteNodePool(organization *v1.Organization, nodePoolID string) error {
	_, hasNodePool := s.nodePools[nodePoolID]
	if !hasNodePool {
		return errors.New("no such node pool")
	}

	return nil
}

func WithClusters(clusters map[string]v1.Cluster) MockOption {
	return func(s *MockClusterService) {
		s.clusters = clusters
	}
}

func WithNodePools(nodePools map[string]v1.NodePool) MockOption {
	return func(s *MockClusterService) {
		s.nodePools = nodePools
	}
}

func WithKubeconfigs(kubeconfigs map[string]clientcmdv1.Config) MockOption {
	return func(s *MockClusterService) {
		s.kubeconfigs = kubeconfigs
	}
}

func NewMockClusterService(mockOptions ...MockOption) *MockClusterService {
	s := MockClusterService{}

	for _, mockOption := range mockOptions {
		mockOption(&s)
	}

	return &s
}

var _ clusterservices.ClusterService = &MockClusterService{}
