package ranchermock

import (
	"net/http"

	"github.com/rancher/norman/clientbase"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type MockOption func(s *managementv3.Client)

func WithSettings(settings map[string]*managementv3.Setting) MockOption {
	mockSetting := MockSetting{
		settings: settings,
	}
	return func(c *managementv3.Client) {
		c.Setting = &mockSetting
	}
}

func WithClusters(clusters map[string]*managementv3.Cluster) MockOption {
	mockCluster := MockCluster{
		clusters: clusters,
	}
	return func(c *managementv3.Client) {
		c.Cluster = &mockCluster
	}
}

func WithNodePools(nodePools map[string]*managementv3.NodePool) MockOption {
	mockNodePool := MockNodePool{
		nodePools: nodePools,
	}
	return func(c *managementv3.Client) {
		c.NodePool = &mockNodePool
	}
}

func WithAPIBaseClient(resources map[string]any) MockOption {
	mockBaseClient := MockBaseClient{
		resources: resources,
	}
	client := http.Client{
		Transport: &mockBaseClient,
	}
	apiBaseClient := clientbase.APIBaseClient{
		Ops: &clientbase.APIOperations{
			Client: &client,
			Opts: &clientbase.ClientOpts{
				TokenKey: "abc123",
			},
			Types: map[string]types.Schema{
				"nodeTemplate": {
					ResourceMethods: []string{
						http.MethodGet,
					},
					Links: map[string]string{
						clientbase.COLLECTION: "http://localhost",
					},
				},
			},
		},
	}
	return func(c *managementv3.Client) {
		c.APIBaseClient = apiBaseClient
	}
}

func WithNodes(nodes map[string]managementv3.Node) MockOption {
	mockNode := MockNode{
		nodes: nodes,
	}
	return func(c *managementv3.Client) {
		c.Node = &mockNode
	}
}

func NewMockRancherClient(mockOptions ...MockOption) *managementv3.Client {
	c := managementv3.Client{}

	for _, mockOption := range mockOptions {
		mockOption(&c)
	}

	if c.Setting == nil {
		c.Setting = &MockSetting{
			settings: map[string]*managementv3.Setting{
				"k8s-versions-current": {
					Value: "v1.2.3",
				},
			},
		}

	}

	return &c
}
