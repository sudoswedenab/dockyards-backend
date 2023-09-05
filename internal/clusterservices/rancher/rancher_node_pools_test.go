package rancher

import (
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/rancher/ranchermock"
	"github.com/google/go-cmp/cmp"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func TestGetNodePool(t *testing.T) {
	tt := []struct {
		name               string
		nodePoolID         string
		ranchermockOptions []ranchermock.MockOption
		expected           v1.NodePool
	}{
		{
			name:       "test simple",
			nodePoolID: "node-pool-123",
			ranchermockOptions: []ranchermock.MockOption{
				ranchermock.WithNodePools(map[string]*managementv3.NodePool{
					"node-pool-123": {
						Resource: types.Resource{
							ID: "node-pool-123",
						},
						ClusterID: "cluster-123",
						Name:      "test",
					},
				}),
				ranchermock.WithNodes(map[string]managementv3.Node{
					"node-123": {
						Resource: types.Resource{
							ID: "node-123",
						},
						NodePoolID: "node-pool-123",
					},
				}),
			},
			expected: v1.NodePool{
				ID:        "node-pool-123",
				ClusterID: "cluster-123",
				Name:      "test",
				Nodes: []v1.Node{
					{
						ID: "node-123",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockRancherClient := ranchermock.NewMockRancherClient(tc.ranchermockOptions...)
			r := rancher{
				managementClient: mockRancherClient,
			}

			actual, err := r.GetNodePool(tc.nodePoolID)
			if err != nil {
				t.Fatalf("error getting node pool: %s", err)
			}

			if !cmp.Equal(actual, &tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(&tc.expected, actual))
			}
		})
	}
}
