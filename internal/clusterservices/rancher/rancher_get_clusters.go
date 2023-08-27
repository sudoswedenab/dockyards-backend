package rancher

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) clusterToModel(cluster *managementv3.Cluster) v1.Cluster {
	createdAt, _ := time.Parse(time.RFC3339, cluster.Created)
	organization, name := names.DecodeName(cluster.Name)

	c := v1.Cluster{
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

func (r *rancher) GetAllClusters() (*[]v1.Cluster, error) {
	clusterCollection, err := r.managementClient.Cluster.ListAll(&types.ListOpts{})
	if err != nil {
		return nil, err
	}

	clusters := []v1.Cluster{}
	for _, cluster := range clusterCollection.Data {
		c := r.clusterToModel(&cluster)
		clusters = append(clusters, c)
	}

	return &clusters, nil
}

func (r *rancher) GetCluster(id string) (*v1.Cluster, error) {
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

	rancherNodePools, err := r.managementClient.NodePool.List(&listOpts)
	if err != nil {
		return nil, err
	}

	for _, rancherNodePool := range rancherNodePools.Data {
		isLoadBalancer := false
		for _, nodeTaint := range rancherNodePool.NodeTaints {
			if nodeTaint.Key == TaintNodeRoleLoadBalancer {
				isLoadBalancer = true
			}
		}

		var customNodeTemplate CustomNodeTemplate
		err := r.managementClient.ByID(managementv3.NodeTemplateType, rancherNodePool.NodeTemplateID, &customNodeTemplate)
		if err != nil {
			return nil, err
		}

		flavorNodePool, err := r.cloudService.GetFlavorNodePool(customNodeTemplate.OpenstackConfig.FlavorID)
		if err != nil {
			return nil, err
		}

		nodePool := v1.NodePool{
			Name:       rancherNodePool.Name,
			Quantity:   int(rancherNodePool.Quantity),
			CPUCount:   flavorNodePool.CPUCount,
			RAMSizeMb:  flavorNodePool.RAMSizeMb,
			DiskSizeGb: flavorNodePool.DiskSizeGb,
		}

		if rancherNodePool.ControlPlane {
			nodePool.ControlPlane = &rancherNodePool.ControlPlane
		}

		if rancherNodePool.Etcd {
			nodePool.Etcd = &rancherNodePool.Etcd
		}

		if !rancherNodePool.Worker {
			nodePool.ControlPlaneComponentsOnly = util.Ptr(true)
		}

		if isLoadBalancer {
			nodePool.LoadBalancer = util.Ptr(true)
		}

		cluster.NodePools = append(cluster.NodePools, nodePool)
	}

	return &cluster, nil
}
