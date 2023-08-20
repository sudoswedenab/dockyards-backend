package rancher

import (
	"errors"
	"reflect"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/cloudmock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/rancher/ranchermock"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func TestGetCluster(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		mockRancherOptions []ranchermock.MockOption
		mockCloudOptions   []cloudmock.MockOption
		expected           model.Cluster
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
					},
				}),
				ranchermock.WithNodePools(map[string]*managementv3.NodePool{
					"node-pool-123": {
						Resource: types.Resource{
							ID: "node-pool-123",
						},
						NodeTemplateID: "node-template-123",
						Worker:         true,
					},
				}),
				ranchermock.WithAPIBaseClient(map[string]any{
					"/node-template-123": &CustomNodeTemplate{
						OpenstackConfig: &openstackConfig{
							FlavorID: "flavor-123",
						},
					},
				}),
			},
			mockCloudOptions: []cloudmock.MockOption{
				cloudmock.WithFlavors(map[string]*model.NodePool{
					"flavor-123": {
						CPUCount:   123,
						RAMSizeMB:  1024,
						DiskSizeGB: 512,
					},
				}),
			},
			expected: model.Cluster{
				ID: "cluster-123",
				NodePools: []model.NodePool{
					{
						CPUCount:   123,
						RAMSizeMB:  1024,
						DiskSizeGB: 512,
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockRancherClient := ranchermock.NewMockRancherClient(tc.mockRancherOptions...)
			mockCloudService := cloudmock.NewMockCloudService(tc.mockCloudOptions...)
			r := rancher{
				managementClient: mockRancherClient,
				cloudService:     mockCloudService,
			}

			actual, err := r.GetCluster(tc.clusterID)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(actual, &tc.expected) {
				t.Errorf("expected %#v, got %#v", &tc.expected, actual)
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
		expected    *[]model.Cluster
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
			expected: &[]model.Cluster{
				{
					ID:           "cluster-123",
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
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("expected %#v, got %#v", tc.expected, actual)
			}
		})
	}
}
