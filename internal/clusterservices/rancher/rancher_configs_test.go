package rancher

import (
	"errors"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/rancher/ranchermock"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func TestClusterOptionsToRKEConfig(t *testing.T) {
	tt := []struct {
		name           string
		clusterOptions model.ClusterOptions
		expected       error
		mockOptions    []ranchermock.MockOption
	}{
		{
			name: "test empty",
		},
		{
			name: "test basic",
			clusterOptions: model.ClusterOptions{
				Name: "basic",
			},
		},
		{
			name: "test supported version",
			clusterOptions: model.ClusterOptions{
				Version: "v1.24.13-rancher2-2",
			},
			mockOptions: []ranchermock.MockOption{
				ranchermock.WithSettings(map[string]*managementv3.Setting{
					"k8s-versions-current": {
						Value: "v1.24.13-rancher2-2",
					},
				}),
			},
		},
		{
			name: "test unsupported version",
			clusterOptions: model.ClusterOptions{
				Version: "v1.24.9-rancher1-1",
			},
			expected: errors.New("unsupported version"),
		},
		{
			name: "test supported ingress provider",
			clusterOptions: model.ClusterOptions{
				IngressProvider: "nginx",
			},
		},
		{
			name: "test unsupported ingress provider",
			clusterOptions: model.ClusterOptions{
				IngressProvider: "traefik",
			},
			expected: errors.New("unsupported ingress provider"),
		},
		{
			name: "test versions error",
			clusterOptions: model.ClusterOptions{
				Version: "v1.2.3",
			},
			mockOptions: []ranchermock.MockOption{
				ranchermock.WithSettings(map[string]*managementv3.Setting{}),
			},
			expected: errors.New("no such setting"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mockRancherClient := ranchermock.NewMockRancherClient(tc.mockOptions...)
			r := rancher{
				managementClient: mockRancherClient,
			}
			_, err := r.clusterOptionsToRKEConfig(&tc.clusterOptions)
			if err != tc.expected {
				if err != nil && tc.expected != nil {
					if err.Error() != tc.expected.Error() {
						t.Errorf("unexpected error testing: got %s,expected:%s", err, tc.expected)
					}
				} else {
					t.Errorf("unexpected error testing: %s", err)
				}

			}
		})
	}
}
