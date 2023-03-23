package rancher

import (
	"errors"
	"testing"

	"bitbucket.org/sudosweden/backend/api/v1/model"
)

func TestClusterOptionsToRKEConfig(t *testing.T) {
	tt := []struct {
		name           string
		clusterOptions model.ClusterOptions
		expected       error
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
				Version: "v1.24.9-rancher1-1",
			},
		},
		{
			name: "test unsupported version",
			clusterOptions: model.ClusterOptions{
				Version: "lalallala",
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
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := Rancher{}
			_, err := r.clusterOptionsToRKEConfig(tc.clusterOptions)
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
