package rancher

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) CreateCluster(organization *v1alpha1.Organization, clusterOptions *v1.ClusterOptions) (*v1.Cluster, error) {
	clusterTemplateName := name.EncodeName(organization.Name, clusterOptions.Name)
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

	clusterName := name.EncodeName(organization.Name, clusterOptions.Name)

	opts := managementv3.Cluster{
		Name:                      clusterName,
		ClusterTemplateRevisionID: createdClusterTemplateRevision.ID,
		ClusterTemplateID:         createdClusterTemplate.ID,
	}

	createdCluster, err := r.managementClient.Cluster.Create(&opts)
	if err != nil {
		return nil, err
	}

	clusterOrganization, clusterName := name.DecodeName(createdCluster.Name)

	cluster := v1.Cluster{
		Organization: clusterOrganization,
		Name:         clusterName,
		Id:           createdCluster.ID,
	}
	return &cluster, nil
}
