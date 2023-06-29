package rancher

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"github.com/rancher/norman/types"
)

func (r *rancher) GetAllClusters() (*[]model.Cluster, error) {
	clusterCollection, err := r.managementClient.Cluster.ListAll(&types.ListOpts{})
	if err != nil {
		return nil, err
	}

	clusters := []model.Cluster{}
	for _, cluster := range clusterCollection.Data {
		createdAt, err := time.Parse(time.RFC3339, cluster.Created)
		if err != nil {
			return nil, err
		}

		organization, name := names.DecodeName(cluster.Name)

		c := model.Cluster{
			Organization: organization,
			Name:         name,
			State:        cluster.State,
			NodeCount:    int(cluster.NodeCount),
			CreatedAt:    createdAt,
			ID:           cluster.ID,
		}
		clusters = append(clusters, c)
	}

	return &clusters, nil
}

func (r *rancher) GetCluster(id string) (*model.Cluster, error) {
	rancherCluster, err := r.managementClient.Cluster.ByID(id)
	if err != nil {
		return nil, err
	}

	organization, name := names.DecodeName(rancherCluster.Name)

	cluster := model.Cluster{
		ID:           rancherCluster.ID,
		Name:         name,
		Organization: organization,
		State:        rancherCluster.State,
		NodeCount:    int(rancherCluster.NodeCount),
	}

	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"clusterId": rancherCluster.ID,
		},
	}

	nodePools, err := r.managementClient.NodePool.List(&listOpts)
	if err != nil {
		return nil, err
	}

	for _, rancherNodePool := range nodePools.Data {
		isLoadBalancer := false
		for _, nodeTaint := range rancherNodePool.NodeTaints {
			if nodeTaint.Key == TaintNodeRoleLoadBalancer {
				isLoadBalancer = true
			}
		}

		nodePool := model.NodePool{
			Name:         rancherNodePool.Name,
			ControlPlane: rancherNodePool.ControlPlane,
			Etcd:         rancherNodePool.Etcd,
			LoadBalancer: isLoadBalancer,
		}

		cluster.NodePools = append(cluster.NodePools, nodePool)
	}

	return &cluster, nil
}
