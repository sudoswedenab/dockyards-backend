package rancher

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) CreateCluster(organization *model.Organization, clusterOptions *model.ClusterOptions) (*model.Cluster, error) {
	clusterTemplateName := names.EncodeName(organization.Name, clusterOptions.Name)
	clusterTemplate := managementv3.ClusterTemplate{
		Name: clusterTemplateName,
	}

	createdClusterTemplate, err := r.managementClient.ClusterTemplate.Create(&clusterTemplate)
	if err != nil {
		return nil, err
	}

	rancherKubernetesEngineConfig, err := r.clusterOptionsToRKEConfig(clusterOptions)
	if err != nil {
		return nil, err
	}
	clusterConfig := managementv3.ClusterSpecBase{
		RancherKubernetesEngineConfig: rancherKubernetesEngineConfig,
	}

	clusterTemplateRevision := managementv3.ClusterTemplateRevision{
		Name:              clusterOptions.Name,
		ClusterTemplateID: createdClusterTemplate.ID,
		ClusterConfig:     &clusterConfig,
	}

	createdClusterTemplateRevision, err := r.managementClient.ClusterTemplateRevision.Create(&clusterTemplateRevision)
	if err != nil {
		return nil, err
	}

	clusterName := names.EncodeName(organization.Name, clusterOptions.Name)

	opts := managementv3.Cluster{
		Name:                      clusterName,
		ClusterTemplateRevisionID: createdClusterTemplateRevision.ID,
		ClusterTemplateID:         createdClusterTemplate.ID,
	}

	createdCluster, err := r.managementClient.Cluster.Create(&opts)
	if err != nil {
		return nil, err
	}

	clusterOrg, clusterName := names.DecodeName(createdCluster.Name)

	cluster := model.Cluster{
		Organization: clusterOrg,
		Name:         clusterName,
		ID:           createdCluster.ID,
	}
	return &cluster, nil
}
