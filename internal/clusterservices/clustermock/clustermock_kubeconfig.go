package clustermock

import (
	"errors"
	"time"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (s *MockClusterService) GetKubeconfig(clusterID string, ttl time.Duration) (*clientcmdapi.Config, error) {
	kubeconfig, hasKubeconfig := s.kubeconfigs[clusterID]
	if !hasKubeconfig {
		return nil, errors.New("no such kubeconfig")
	}

	return &kubeconfig, nil
}
