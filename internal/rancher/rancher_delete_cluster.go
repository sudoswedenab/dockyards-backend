package rancher

import (
	"errors"

	"github.com/rancher/norman/types"
)

func (r *Rancher) DeleteCluster(name string) error {
	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"name": name,
		},
	}
	clusterCollection, err := r.ManagementClient.Cluster.List(&listOpts)
	if err != nil {
		return err
	}

	r.Logger.Debug("list cluster collection", "len", len(clusterCollection.Data))

	for _, cluster := range clusterCollection.Data {
		if cluster.Name == name {
			r.Logger.Debug("cluster to delete found", "id", cluster.ID, "name", cluster.Name)

			err := r.ManagementClient.Cluster.Delete(&cluster)
			if err != nil {
				return err
			}
			return nil
		}
	}

	return errors.New("unable to find cluster to delete")
}
