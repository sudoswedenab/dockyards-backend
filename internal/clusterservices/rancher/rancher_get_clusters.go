package rancher

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) clusterToModel(cluster *managementv3.Cluster) model.Cluster {
	createdAt, _ := time.Parse(time.RFC3339, cluster.Created)
	organization, name := names.DecodeName(cluster.Name)

	c := model.Cluster{
		Organization: organization,
		Name:         name,
		State:        cluster.State,
		NodeCount:    int(cluster.NodeCount),
		CreatedAt:    createdAt,
		ID:           cluster.ID,
	}

	if cluster.RancherKubernetesEngineConfig != nil {
		c.Version = cluster.RancherKubernetesEngineConfig.Version
	}

	return c
}

func (r *rancher) GetAllClusters() (*[]model.Cluster, error) {
	clusterCollection, err := r.managementClient.Cluster.ListAll(&types.ListOpts{})
	if err != nil {
		return nil, err
	}

	clusters := []model.Cluster{}
	for _, cluster := range clusterCollection.Data {
		c := r.clusterToModel(&cluster)
		clusters = append(clusters, c)
	}

	return &clusters, nil
}

func (r *rancher) GetCluster(id string) (*model.Cluster, error) {
	rancherCluster, err := r.managementClient.Cluster.ByID(id)
	if err != nil {
		return nil, err
	}

	cluster := r.clusterToModel(rancherCluster)

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
			Name:                       rancherNodePool.Name,
			ControlPlane:               rancherNodePool.ControlPlane,
			Etcd:                       rancherNodePool.Etcd,
			LoadBalancer:               isLoadBalancer,
			Quantity:                   int(rancherNodePool.Quantity),
			ControlPlaneComponentsOnly: !rancherNodePool.Worker,
		}

		cluster.NodePools = append(cluster.NodePools, nodePool)
	}

	return &cluster, nil
}
