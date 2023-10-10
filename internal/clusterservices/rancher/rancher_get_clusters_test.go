package rancher

import (
	"errors"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/rancher/ranchermock"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetCluster(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		mockRancherOptions []ranchermock.MockOption
		expected           v1alpha1.ClusterStatus
	}{
		{
			name:      "test simple",
			clusterID: "cluster-123",
			mockRancherOptions: []ranchermock.MockOption{
				ranchermock.WithClusters(map[string]*managementv3.Cluster{
					"cluster-123": {
						Resource: types.Resource{
							ID: "cluster-123",
						},
						Conditions: []managementv3.ClusterCondition{
							{
								Type:   "Provisioned",
								Status: "True",
							},
							{
								Type:   "Ready",
								Status: "True",
							},
						},
						State: "testing",
					},
				}),
			},
			expected: v1alpha1.ClusterStatus{
				ClusterServiceID: "cluster-123",
				Conditions: []metav1.Condition{
					{
						Type:    v1alpha1.ReadyCondition,
						Status:  metav1.ConditionTrue,
						Reason:  v1alpha1.ClusterReadyReason,
						Message: "testing",
					},
				},
			},
		},
		{
			name:      "test provisioning cluster",
			clusterID: "cluster-123",
			mockRancherOptions: []ranchermock.MockOption{
				ranchermock.WithClusters(map[string]*managementv3.Cluster{
					"cluster-123": {
						Resource: types.Resource{
							ID: "cluster-123",
						},
						Conditions: []managementv3.ClusterCondition{
							{
								Type:    "Provisioned",
								Status:  "False",
								Message: "provisioning etcd",
							},
							{
								Type:   "Ready",
								Status: "False",
							},
						},
						State: "testing",
					},
				}),
			},
			expected: v1alpha1.ClusterStatus{
				ClusterServiceID: "cluster-123",
				Conditions: []metav1.Condition{
					{
						Type:    v1alpha1.ReadyCondition,
						Status:  metav1.ConditionFalse,
						Reason:  v1alpha1.ClusterReadyReason,
						Message: "testing",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockRancherClient := ranchermock.NewMockRancherClient(tc.mockRancherOptions...)

			r := rancher{
				managementClient: mockRancherClient,
			}

			actual, err := r.GetCluster(tc.clusterID)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !cmp.Equal(actual, &tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(&tc.expected, actual))
			}
		})
	}
}

func TestGetClusterErrors(t *testing.T) {
	tt := []struct {
		name        string
		clusterID   string
		mockOptions []ranchermock.MockOption
		expected    error
	}{
		{
			name:      "test missing",
			clusterID: "cluster-123",
			mockOptions: []ranchermock.MockOption{
				ranchermock.WithClusters(map[string]*managementv3.Cluster{
					"cluster-234": {},
				}),
			},
			expected: errors.New("no such cluster"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockRancherClient := ranchermock.NewMockRancherClient(tc.mockOptions...)
			r := rancher{
				managementClient: mockRancherClient,
			}
			_, err := r.GetCluster(tc.clusterID)
			if err != nil && err.Error() != tc.expected.Error() {
				t.Errorf("expected error %s, got %s", tc.expected, err)
			}

		})
	}
}

func TestGetAllClusters(t *testing.T) {
	tt := []struct {
		name        string
		mockOptions []ranchermock.MockOption
		expected    *[]v1.Cluster
	}{
		{
			name: "test single",
			mockOptions: []ranchermock.MockOption{
				ranchermock.WithClusters(map[string]*managementv3.Cluster{
					"cluster-123": {
						Resource: types.Resource{
							ID: "cluster-123",
						},
						Name: "test-cluster",
						RancherKubernetesEngineConfig: &managementv3.RancherKubernetesEngineConfig{
							Version: "v1.2.3",
						},
					},
				}),
			},
			expected: &[]v1.Cluster{
				{
					Id:           "cluster-123",
					Organization: "test",
					Name:         "cluster",
					Version:      "v1.2.3",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockRancherClient := ranchermock.NewMockRancherClient(tc.mockOptions...)
			r := rancher{
				managementClient: mockRancherClient,
			}
			actual, err := r.GetAllClusters()
			if err != nil {
				t.Fatalf("unxepected error getting clusters: %s", err)
			}
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
