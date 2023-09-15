package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/rancher/norman/types"
)

func (r *rancher) DeleteCluster(organization *v1.Organization, cluster *v1.Cluster) error {
	encodedName := name.EncodeName(cluster.Organization, cluster.Name)

	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"name": encodedName,
		},
	}
	clusterCollection, err := r.managementClient.Cluster.List(&listOpts)
	if err != nil {
		return err
	}

	r.logger.Debug("list cluster collection", "len", len(clusterCollection.Data))

	for _, cluster := range clusterCollection.Data {
		if cluster.Name == encodedName {
			r.logger.Debug("cluster to delete found", "id", cluster.ID, "name", cluster.Name)

			err := r.deleteNodePools(organization, cluster.ID)
			if err != nil {
				// any errors here are only logged as warnings, they do not abort the cluster deletion
				// deleting any objects related to the node pools are not required to delete the cluster
				r.logger.Warn("error deleting node pool or resources", "err", err)
			}

			err = r.managementClient.Cluster.Delete(&cluster)
			if err != nil {
				return err
			}
			r.logger.Debug("deleted cluster", "id", cluster.ID, "name", cluster.Name)

			clusterTemplate, err := r.managementClient.ClusterTemplate.ByID(cluster.ClusterTemplateID)
			if err != nil {
				r.logger.Warn("error fetching cluster template by id", "id", cluster.ClusterTemplateID)
			}

			// cluster template cannot be deleted at this point
			// add it to the garbage
			r.addGarbage(&clusterTemplate.Resource)

			r.logger.Debug("added cluster template to garbage", "id", clusterTemplate.ID)

			return nil
		}
	}

	return errors.New("unable to find cluster to delete")
}

func (r *rancher) deleteNodePools(organization *v1.Organization, clusterID string) error {
	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"clusterId": clusterID,
		},
	}

	nodePoolCollection, err := r.managementClient.NodePool.List(&listOpts)
	if err != nil {
		return err
	}

	for _, nodePool := range nodePoolCollection.Data {
		err := r.DeleteNodePool(organization, nodePool.ID)
		if err != nil {
			r.logger.Warn("error deleting node pool", "id", nodePool.ID, "err", err)
		}
	}

	return nil
}
