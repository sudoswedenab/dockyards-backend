package rancher

import (
	"strings"
	"time"

	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"
)

func (r *rancher) GetKubeconfig(clusterID string, ttl time.Duration) (string, error) {
	rancherCluster, err := r.managementClient.Cluster.ByID(clusterID)
	if err != nil {
		return "", err
	}

	r.logger.Debug("generating kubeconfig for cluster", "id", rancherCluster.ID)

	generatedKubeConfig, err := r.managementClient.Cluster.ActionGenerateKubeconfig(rancherCluster)
	if err != nil {
		return "", err
	}

	r.logger.Debug("generated kubeconfig for cluster", "id", rancherCluster.ID)

	var config v1.Config
	err = yaml.Unmarshal([]byte(generatedKubeConfig.Config), &config)
	if err != nil {
		return "", err
	}

	authInfo := config.AuthInfos[0].AuthInfo

	tokenID := authInfo.Token[:strings.Index(authInfo.Token, ":")]

	unrestrictedToken, err := r.managementClient.Token.ByID(tokenID)
	if err != nil {
		return "", err
	}

	limitedToken := managementv3.Token{
		ClusterID: clusterID,
		TTLMillis: ttl.Milliseconds(),
	}

	r.logger.Debug("creating limited token", "ttl", ttl)

	createdToken, err := r.managementClient.Token.Create(&limitedToken)
	if err != nil {
		return "", err
	}

	r.logger.Debug("created limited token", "id", createdToken.ID, "ttl", ttl)

	authInfo.Token = createdToken.Token
	config.AuthInfos[0].Name = clusterID
	config.AuthInfos[0].AuthInfo = authInfo
	config.Clusters[0].Name = clusterID
	config.Contexts[0].Name = clusterID
	config.Contexts[0].Context.Cluster = clusterID
	config.Contexts[0].Context.AuthInfo = clusterID
	config.CurrentContext = clusterID

	r.logger.Debug("deleting unrestricted token", "id", unrestrictedToken.ID)

	err = r.managementClient.Token.Delete(unrestrictedToken)
	if err != nil {
		return "", err
	}

	r.logger.Debug("deleted unrestricted token", "id", unrestrictedToken.ID)

	b, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
