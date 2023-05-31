package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	normanTypes "github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) DeleteCluster(cluster *model.Cluster) error {
	encodedName := names.EncodeName(cluster.Organization, cluster.Name)

	listOpts := normanTypes.ListOpts{
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

			err := r.deleteNodePools(cluster.ID)
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

func (r *rancher) deleteNodePools(clusterID string) error {
	listOpts := normanTypes.ListOpts{
		Filters: map[string]interface{}{
			"clusterId": clusterID,
		},
	}

	nodePoolCollection, err := r.managementClient.NodePool.List(&listOpts)
	if err != nil {
		return err
	}

	r.logger.Debug("node pools listed", "len", len(nodePoolCollection.Data))
	for _, nodePool := range nodePoolCollection.Data {
		r.logger.Debug("node pool", "id", nodePool.ID, "cluster", nodePool.ClusterID,
			"node_template_id", nodePool.NodeTemplateID)

		var customNodeTemplate CustomNodeTemplate
		err := r.managementClient.ByID(managementv3.NodeTemplateType, nodePool.NodeTemplateID, &customNodeTemplate)
		if err != nil {
			r.logger.Warn("error fetching node template by id", "id", nodePool.NodeTemplateID)
			return err
		}

		r.logger.Debug("node pool custom node template", "id", customNodeTemplate.ID)

		cloudConfig := types.CloudConfig{
			KeypairName: customNodeTemplate.OpenstackConfig.KeypairName,
		}

		err = r.cloudService.CleanEnvironment(&cloudConfig)
		if err != nil {
			r.logger.Warn("error cleaning openstack environment", "err", err)
			return err
		}

		err = r.managementClient.NodePool.Delete(&nodePool)
		if err != nil {
			r.logger.Warn("error deleting node pool", "err", err)
			return err
		}
		r.logger.Debug("deleted node pool", "id", nodePool.ID, "name", nodePool.Name)

		// node template cannot be deleted at this point
		// add it to the garbage
		r.addGarbage(&customNodeTemplate.NodeTemplate.Resource)

		r.logger.Debug("added node template to garbage", "id", customNodeTemplate.ID)
	}

	return nil
}
