package rancher

import (
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) RancherCreateCluster(dockerRootDir, name, ctrId, ctId string) (managementv3.Cluster, error) {
	cluster := managementv3.Cluster{
		DockerRootDir:             dockerRootDir,
		Name:                      name,
		ClusterTemplateRevisionID: ctrId,
		ClusterTemplateID:         ctId,
	}

	createdCluster, err := r.ManagementClient.Cluster.Create(&cluster)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	return *createdCluster, nil
}
