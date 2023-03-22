package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) RancherCreateCluster(clusterOptions model.ClusterOptions) (managementv3.Cluster, error) {

	clusterTemplate := managementv3.ClusterTemplate{
		Name: "testar",
	}

	createdClusterTemplate, err := r.ManagementClient.ClusterTemplate.Create(&clusterTemplate)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	rancherKubernetesEngineConfig, err := r.clusterOptionsToRKEConfig(clusterOptions)
	if err != nil {
		return managementv3.Cluster{}, err
	}
	clusterConfig := managementv3.ClusterSpecBase{
		RancherKubernetesEngineConfig: rancherKubernetesEngineConfig,
	}

	clusterTemplateRevision := managementv3.ClusterTemplateRevision{
		Name:              clusterOptions.Name,
		ClusterTemplateID: createdClusterTemplate.ID,
		ClusterConfig:     &clusterConfig,
	}

	createdClusterTemplateRevision, err := r.ManagementClient.ClusterTemplateRevision.Create(&clusterTemplateRevision)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	cluster := managementv3.Cluster{
		Name:                      clusterOptions.Name,
		ClusterTemplateRevisionID: createdClusterTemplateRevision.ID,
		ClusterTemplateID:         createdClusterTemplate.ID,
	}

	createdCluster, err := r.ManagementClient.Cluster.Create(&cluster)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	return *createdCluster, nil
}
