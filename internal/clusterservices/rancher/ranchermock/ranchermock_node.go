package ranchermock

import (
	"errors"

	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockNode struct {
	managementv3.NodeOperations
	nodes map[string]managementv3.Node
}

func (n *MockNode) ListAll(opts *types.ListOpts) (*managementv3.NodeCollection, error) {
	nodePoolID, hasNodePoolID := opts.Filters["nodePoolId"]
	if !hasNodePoolID {
		return nil, errors.New("no node pool id in opts")
	}

	nodeCollection := managementv3.NodeCollection{}
	for _, node := range n.nodes {
		if node.NodePoolID == nodePoolID {
			nodeCollection.Data = append(nodeCollection.Data, node)
		}
	}

	return &nodeCollection, nil
}
