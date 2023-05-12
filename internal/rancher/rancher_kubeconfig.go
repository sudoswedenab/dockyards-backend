package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/rancher/norman/types"
)

func (r *rancher) GetKubeConfig(cluster *model.Cluster) (string, error) {
	encodedName := encodeName(cluster.Organization, cluster.Name)

	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"name": encodedName,
		},
	}
	clusterCollection, err := r.managementClient.Cluster.List(&listOpts)
	if err != nil {
		return "", err
	}

	r.logger.Debug("list cluster collection", "len", len(clusterCollection.Data))

	for _, data := range clusterCollection.Data {
		if data.Name == encodedName {
			r.logger.Debug("cluster to generate kubeconfig for found", "id", data.ID, "name", data.Name)
			generatedKubeConfig, err := r.managementClient.Cluster.ActionGenerateKubeconfig(&data)
			if err != nil {
				return "", err
			}
			return generatedKubeConfig.Config, nil
		}
	}

	return "", errors.New("no such cluster")
}
