package ranchermock

import (
	"errors"

	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockNodePool struct {
	managementv3.NodePoolOperations
	nodePools map[string]*managementv3.NodePool
}

func (p *MockNodePool) List(opts *types.ListOpts) (*managementv3.NodePoolCollection, error) {
	nodePoolCollection := managementv3.NodePoolCollection{}

	for _, nodePool := range p.nodePools {
		nodePoolCollection.Data = append(nodePoolCollection.Data, *nodePool)
	}

	return &nodePoolCollection, nil
}

func (p *MockNodePool) ByID(nodePoolID string) (*managementv3.NodePool, error) {
	nodePool, hasNodePool := p.nodePools[nodePoolID]
	if !hasNodePool {
		return nil, errors.New("no such node pool")
	}

	return nodePool, nil
}

var _ managementv3.NodePoolOperations = &MockNodePool{}
