package rancher

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/rancher/norman/types"
)

func (r *Rancher) GetAllClusters() (*[]model.Cluster, error) {
	clusterCollection, err := r.ManagementClient.Cluster.ListAll(&types.ListOpts{})
	if err != nil {
		return nil, err
	}

	clusters := []model.Cluster{}
	for _, cluster := range clusterCollection.Data {
		createdAt, err := time.Parse(time.RFC3339, cluster.Created)
		if err != nil {
			return nil, err
		}

		organization, name := decodeName(cluster.Name)

		c := model.Cluster{
			Organization: organization,
			Name:         name,
			State:        cluster.State,
			NodeCount:    int(cluster.NodeCount),
			CreatedAt:    createdAt,
		}
		clusters = append(clusters, c)
	}

	return &clusters, nil
}
