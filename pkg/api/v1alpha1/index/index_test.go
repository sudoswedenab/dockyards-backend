package index_test

import (
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIndexByClusterServiceID(t *testing.T) {
	tt := []struct {
		name     string
		object   client.Object
		expected []string
	}{
		{
			name: "test cluster",
			object: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					ClusterServiceID: "cb2b1a95-8923-422c-a379-9a06e792e6a8",
				},
			},
			expected: []string{
				"cb2b1a95-8923-422c-a379-9a06e792e6a8",
			},
		},
		{
			name: "test nodepool",
			object: &v1alpha1.NodePool{
				Status: v1alpha1.NodePoolStatus{
					ClusterServiceID: "d9f849d2-2b26-4a18-a06c-391d9030a2a1",
				},
			},
			expected: []string{
				"d9f849d2-2b26-4a18-a06c-391d9030a2a1",
			},
		},
		{
			name: "test node",
			object: &v1alpha1.Node{
				Status: v1alpha1.NodeStatus{
					ClusterServiceID: "361d4f33-7b8d-4b15-94d6-1d00a693c4f8",
				},
			},
			expected: []string{
				"361d4f33-7b8d-4b15-94d6-1d00a693c4f8",
			},
		},
		{
			name:   "test configmap",
			object: &corev1.ConfigMap{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := index.IndexByClusterServiceID(tc.object)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
