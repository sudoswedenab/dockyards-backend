package rancher

import (
	"errors"
	"time"

	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
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

			nodeTemplatesToDelete, err := r.deleteNodePools(cluster.ID)
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

			// now when the cluster has been deleted we can delete any node templates that was used
			// same error handling as before, only log warnings
			for _, nodeTemplateID := range nodeTemplatesToDelete {
				// try deleting the node template a maximum of ten times
				for i := 1; i < 11; i++ {
					r.Logger.Debug("fetching node template to delete", "id", nodeTemplateID, "attempt", i)

					nodeTemplate, err := r.ManagementClient.NodeTemplate.ByID(nodeTemplateID)
					if err != nil {
						r.Logger.Warn("error fetching node template by id", "id", nodeTemplateID, "err", err)
						continue
					}

					r.Logger.Debug("node template to delete", "id", nodeTemplate.ID, "name", nodeTemplate.Name, "links", nodeTemplate.Links)

					// node templates have status and transitioning fields but they are always active, even on unused node templates
					// reliable method seems to be to check the links if there is one for the remove action
					_, hasRemoveLink := nodeTemplate.Links["remove"]
					if !hasRemoveLink {
						time.Sleep(time.Second)
						continue
					}

					err = r.ManagementClient.NodeTemplate.Delete(nodeTemplate)
					if err != nil {
						r.Logger.Warn("error deleting node template", "id", nodeTemplate.ID, "name", nodeTemplate.Name, "err", err)
						continue
					}

					r.Logger.Debug("deleted node template", "id", nodeTemplate.ID, "name", nodeTemplate.Name)
					break
				}
			}

			return nil
		}
	}

	return errors.New("unable to find cluster to delete")
}

func (r *Rancher) deleteNodePools(clusterID string) ([]string, error) {
	listOpts := types.ListOpts{
		Filters: map[string]interface{}{
			"clusterId": clusterID,
		},
	}

	nodePoolCollection, err := r.ManagementClient.NodePool.List(&listOpts)
	if err != nil {
		return []string{}, err
	}

	// keep a list of node template to delete in the future
	// node templates can not be deleted immediately after the node pool
	// until the underlying infrastructure is deleted the node template remains active
	nodeTemplateIDs := []string{}

	r.Logger.Debug("node pools listed", "len", len(nodePoolCollection.Data))
	for _, nodePool := range nodePoolCollection.Data {
		r.Logger.Debug("node pool", "id", nodePool.ID, "cluster", nodePool.ClusterID,
			"node_template_id", nodePool.NodeTemplateID)

		var customNodeTemplate CustomNodeTemplate
		err := r.ManagementClient.ByID(managementv3.NodeTemplateType, nodePool.NodeTemplateID, &customNodeTemplate)
		if err != nil {
			r.Logger.Warn("error fetching node template by id", "id", nodePool.NodeTemplateID)
			return nodeTemplateIDs, err
		}

		r.Logger.Debug("custom node template", "id", customNodeTemplate.ID, "openstackConfig",
			customNodeTemplate.OpenstackConfig)

		err = r.cleanOpenstackEnvironment(customNodeTemplate.OpenstackConfig)
		if err != nil {
			r.Logger.Warn("error cleaning openstack environment", "err", err)
			return nodeTemplateIDs, err
		}

		err = r.ManagementClient.NodePool.Delete(&nodePool)
		if err != nil {
			r.Logger.Warn("error deleting node pool", "err", err)
			return nodeTemplateIDs, err
		}
		r.Logger.Debug("deleted node pool", "id", nodePool.ID, "name", nodePool.Name)

		nodeTemplateIDs = append(nodeTemplateIDs, customNodeTemplate.NodeTemplate.ID)
	}

	// return the list of node templates that needs to be deleted by other logic
	return nodeTemplateIDs, nil
}
