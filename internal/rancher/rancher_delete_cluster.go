package rancher

import managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"

func (r *Rancher) DeleteCluster(container managementv3.Cluster) error {
	err := r.ManagementClient.Cluster.Delete(&container)

	if err != nil {
		return err
	}

	return nil
}
