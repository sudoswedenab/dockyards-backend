package rancher

import (
	"strings"
	"time"

	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (r *rancher) GetKubeconfig(clusterID string, ttl time.Duration) (*clientcmdapi.Config, error) {
	rancherCluster, err := r.managementClient.Cluster.ByID(clusterID)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("generating kubeconfig for cluster", "id", rancherCluster.ID)

	generatedKubeConfig, err := r.managementClient.Cluster.ActionGenerateKubeconfig(rancherCluster)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("generated kubeconfig for cluster", "id", rancherCluster.ID)

	config, err := clientcmd.Load([]byte(generatedKubeConfig.Config))
	if err != nil {
		return nil, err
	}

	currentUser := config.AuthInfos[config.CurrentContext]

	tokenID := currentUser.Token[:strings.Index(currentUser.Token, ":")]

	unrestrictedToken, err := r.managementClient.Token.ByID(tokenID)
	if err != nil {
		return nil, err
	}

	limitedToken := managementv3.Token{
		ClusterID: clusterID,
		TTLMillis: ttl.Milliseconds(),
	}

	r.logger.Debug("creating limited token", "ttl", ttl)

	createdToken, err := r.managementClient.Token.Create(&limitedToken)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("created limited token", "id", createdToken.ID, "ttl", ttl)

	currentUser.Token = createdToken.Token
	config.AuthInfos[config.CurrentContext] = currentUser

	r.logger.Debug("deleting unrestricted token", "id", unrestrictedToken.ID)

	err = r.managementClient.Token.Delete(unrestrictedToken)
	if err != nil {
		return nil, err
	}

	r.logger.Debug("deleted unrestricted token", "id", unrestrictedToken.ID)

	return config, nil
}
