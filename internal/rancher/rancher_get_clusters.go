package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/rancher/norman/types"
)

func (r *Rancher) GetAllClusters() (*[]model.Cluster, error) {
	clusterCollection, err := r.ManagementClient.Cluster.ListAll(&types.ListOpts{})
	if err != nil {
		return nil, err
	}

	clusters := []model.Cluster{}
	for _, cluster := range clusterCollection.Data {
		c := model.Cluster{
			Name: cluster.Name,
		}
		clusters = append(clusters, c)
	}

	return &clusters, nil
}
