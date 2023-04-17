package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) DeleteCluster(cluster *model.Cluster) error {
	encodedName := encodeName(cluster.Organization, cluster.Name)

	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"name": encodedName,
		},
	}
	clusterCollection, err := r.ManagementClient.Cluster.List(&listOpts)
	if err != nil {
		return err
	}

	r.Logger.Debug("list cluster collection", "len", len(clusterCollection.Data))

	for _, cluster := range clusterCollection.Data {
		if cluster.Name == encodedName {
			r.Logger.Debug("cluster to delete found", "id", cluster.ID, "name", cluster.Name)

			err := r.deleteNodePools(cluster.ID)
			if err != nil {
				// any errors here are only logged as warnings, they do not abort the cluster deletion
				// deleting any objects related to the node pools are not required to delete the cluster
				r.Logger.Warn("error deleting node pool or resources", "err", err)
			}

			err = r.ManagementClient.Cluster.Delete(&cluster)
			if err != nil {
				return err
			}
			r.Logger.Debug("deleted cluster", "id", cluster.ID, "name", cluster.Name)

			clusterTemplate, err := r.ManagementClient.ClusterTemplate.ByID(cluster.ClusterTemplateID)
			if err != nil {
				r.Logger.Warn("error fetching cluster template by id", "id", cluster.ClusterTemplateID)
			}

			// cluster template cannot be deleted at this point
			// add it to the garbage
			r.addGarbage(&clusterTemplate.Resource)

			r.Logger.Debug("added cluster template to garbage", "id", clusterTemplate.ID)

			return nil
		}
	}

	return errors.New("unable to find cluster to delete")
}

func (r *Rancher) deleteNodePools(clusterID string) error {
	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"clusterId": clusterID,
		},
	}

	nodePoolCollection, err := r.ManagementClient.NodePool.List(&listOpts)
	if err != nil {
		return err
	}

	r.Logger.Debug("node pools listed", "len", len(nodePoolCollection.Data))
	for _, nodePool := range nodePoolCollection.Data {
		r.Logger.Debug("node pool", "id", nodePool.ID, "cluster", nodePool.ClusterID,
			"node_template_id", nodePool.NodeTemplateID)

		var customNodeTemplate CustomNodeTemplate
		err := r.ManagementClient.ByID(managementv3.NodeTemplateType, nodePool.NodeTemplateID, &customNodeTemplate)
		if err != nil {
			r.Logger.Warn("error fetching node template by id", "id", nodePool.NodeTemplateID)
			return err
		}

		r.Logger.Debug("custom node template", "id", customNodeTemplate.ID, "openstackConfig",
			customNodeTemplate.OpenstackConfig)

		err = r.cleanOpenstackEnvironment(customNodeTemplate.OpenstackConfig)
		if err != nil {
			r.Logger.Warn("error cleaning openstack environment", "err", err)
			return err
		}

		err = r.ManagementClient.NodePool.Delete(&nodePool)
		if err != nil {
			r.Logger.Warn("error deleting node pool", "err", err)
			return err
		}
		r.Logger.Debug("deleted node pool", "id", nodePool.ID, "name", nodePool.Name)

		// node template cannot be deleted at this point
		// add it to the garbage
		r.addGarbage(&customNodeTemplate.NodeTemplate.Resource)

		r.Logger.Debug("added node template to garbage", "id", customNodeTemplate.ID)
	}

	return nil
}
