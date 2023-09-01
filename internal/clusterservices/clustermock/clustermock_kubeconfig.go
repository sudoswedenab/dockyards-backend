package clustermock

import (
	"errors"
	"time"

	"k8s.io/client-go/tools/clientcmd/api/v1"
)

func (s *MockClusterService) GetKubeconfig(clusterID string, ttl time.Duration) (*v1.Config, error) {
	kubeconfig, hasKubeconfig := s.kubeconfigs[clusterID]
	if !hasKubeconfig {
		return nil, errors.New("no such kubeconfig")
	}

	return &kubeconfig, nil
}
