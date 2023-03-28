package rancher

import (
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) GetClusters(opts *types.ListOpts) (managementv3.ClusterCollection, error) {
	clusters, err := r.ManagementClient.Cluster.List(opts)
	if err != nil {
		return managementv3.ClusterCollection{}, err
	}

	return *clusters, nil
}
