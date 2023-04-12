package rancher

import managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"

func (r *Rancher) DeleteCluster(name string) error {
	cluster := managementv3.Cluster{
		Name: name,
	}

	err := r.ManagementClient.Cluster.Delete(&cluster)
	if err != nil {
		return err
	}

	return nil
}
