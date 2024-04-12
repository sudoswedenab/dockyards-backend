package deployment

import (
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func TestAddNormalizedName(t *testing.T) {
	tt := []struct {
		name       string
		deployment v1.Deployment
		expected   v1.Deployment
	}{
		{
			name: "test container image",
			deployment: v1.Deployment{
				ContainerImage: util.Ptr("test:v1.2.3"),
			},
			expected: v1.Deployment{
				Name:           util.Ptr("test"),
				Namespace:      util.Ptr("test"),
				ContainerImage: util.Ptr("docker.io/library/test:v1.2.3"),
				Type:           v1.DeploymentTypeContainerImage,
			},
		},
		{
			name: "test helm chart",
			deployment: v1.Deployment{
				HelmChart: util.Ptr("test"),
			},
			expected: v1.Deployment{
				Name:      util.Ptr("test"),
				Namespace: util.Ptr("test"),
				HelmChart: util.Ptr("test"),
				Type:      v1.DeploymentTypeHelm,
			},
		},
		{
			name: "test kustomize",
			deployment: v1.Deployment{
				Kustomize: util.Ptr(map[string][]byte{
					"kustomization.yaml": []byte("test"),
				}),
			},
			expected: v1.Deployment{
				Name:      util.Ptr(""),
				Namespace: util.Ptr(""),
				Kustomize: util.Ptr(map[string][]byte{
					"kustomization.yaml": []byte("test"),
				}),
				Type: v1.DeploymentTypeKustomize,
			},
		},
	}

	ignoreTypes := cmpopts.IgnoreTypes(uuid.UUID{})

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := AddNormalizedName(&tc.deployment)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !cmp.Equal(tc.deployment, tc.expected, ignoreTypes) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, tc.deployment, ignoreTypes))
			}

		})
	}
}

func TestAddNormalizedNameErrors(t *testing.T) {
	tt := []struct {
		name       string
		deployment v1.Deployment
	}{
		{
			name: "test invalid container image",
			deployment: v1.Deployment{
				ContainerImage: util.Ptr("http://localhost/nginx:v1.2.3"),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := AddNormalizedName(&tc.deployment)

			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
