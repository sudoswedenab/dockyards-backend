package ranchermock

import (
	"errors"

	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockCluster struct {
	managementv3.ClusterOperations
	clusters map[string]*managementv3.Cluster
}

func (c *MockCluster) ByID(id string) (*managementv3.Cluster, error) {
	cluster, hasCluster := c.clusters[id]
	if !hasCluster {
		return nil, errors.New("no such cluster")
	}

	return cluster, nil
}

func (c *MockCluster) ListAll(opts *types.ListOpts) (*managementv3.ClusterCollection, error) {
	clusterCollection := managementv3.ClusterCollection{}

	for _, cluster := range c.clusters {
		clusterCollection.Data = append(clusterCollection.Data, *cluster)
	}

	return &clusterCollection, nil
}
