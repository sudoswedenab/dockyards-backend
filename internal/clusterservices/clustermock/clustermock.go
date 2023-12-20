package clustermock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type MockClusterService struct {
	clusterservices.ClusterService
	clusters       map[string]v1.Cluster
	nodePoolStatus map[string]v1alpha1.NodePoolStatus
	kubeconfigs    map[string]clientcmdapi.Config
}

type MockOption func(*MockClusterService)

func (s *MockClusterService) GetAllClusters() (*[]v1.Cluster, error) {
	clusters := []v1.Cluster{}

	for _, cluster := range s.clusters {
		clusters = append(clusters, cluster)
	}

	return &clusters, nil
}

func (s *MockClusterService) CreateNodePool(organization *v1alpha2.Organization, cluster *v1alpha1.Cluster, nodePool *v1alpha1.NodePool) (*v1alpha1.NodePoolStatus, error) {
	nodePoolStatus := v1alpha1.NodePoolStatus{
		ClusterServiceID: "node-pool-123",
		Resources:        nodePool.Spec.Resources,
	}

	return &nodePoolStatus, nil
}

func (s *MockClusterService) DeleteCluster(organization *v1alpha2.Organization, cluster *v1.Cluster) error {
	for _, c := range s.clusters {
		if c.Organization == organization.Name && c.Name == cluster.Name {
			return nil
		}
	}

	return errors.New("no such cluster")

}

func (s *MockClusterService) GetCluster(clusterID string) (*v1alpha1.ClusterStatus, error) {
	cluster, hasCluster := s.clusters[clusterID]
	if !hasCluster {
		return nil, errors.New("no such cluster")
	}

	clusterStatus := v1alpha1.ClusterStatus{
		ClusterServiceID: cluster.Id,
	}

	return &clusterStatus, nil
}

func (s *MockClusterService) CollectMetrics() error {
	return nil
}

func (s *MockClusterService) DeleteGarbage() {
	return
}

func (s *MockClusterService) GetNodePool(nodePoolID string) (*v1alpha1.NodePoolStatus, error) {
	nodePoolStatus, hasNodePoolStatus := s.nodePoolStatus[nodePoolID]
	if !hasNodePoolStatus {
		return nil, errors.New("no such node pool")
	}

	return &nodePoolStatus, nil
}

func (s *MockClusterService) DeleteNodePool(organization *v1alpha2.Organization, nodePoolID string) error {
	_, hasNodePoolStatus := s.nodePoolStatus[nodePoolID]
	if !hasNodePoolStatus {
		return errors.New("no such node pool")
	}

	return nil
}

func WithClusters(clusters map[string]v1.Cluster) MockOption {
	return func(s *MockClusterService) {
		s.clusters = clusters
	}
}

func WithNodePools(nodePoolStatus map[string]v1alpha1.NodePoolStatus) MockOption {
	return func(s *MockClusterService) {
		s.nodePoolStatus = nodePoolStatus
	}
}

func WithKubeconfigs(kubeconfigs map[string]clientcmdapi.Config) MockOption {
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
