package deployment

import (
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"k8s.io/utils/ptr"
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
				ContainerImage: ptr.To("test:v1.2.3"),
			},
			expected: v1.Deployment{
				Name:           ptr.To("test"),
				Namespace:      ptr.To("test"),
				ContainerImage: ptr.To("docker.io/library/test:v1.2.3"),
				Type:           v1.DeploymentTypeContainerImage,
			},
		},
		{
			name: "test helm chart",
			deployment: v1.Deployment{
				HelmChart: ptr.To("test"),
			},
			expected: v1.Deployment{
				Name:      ptr.To("test"),
				Namespace: ptr.To("test"),
				HelmChart: ptr.To("test"),
				Type:      v1.DeploymentTypeHelm,
			},
		},
		{
			name: "test kustomize",
			deployment: v1.Deployment{
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("test"),
				}),
			},
			expected: v1.Deployment{
				Name:      ptr.To(""),
				Namespace: ptr.To(""),
				Kustomize: ptr.To(map[string][]byte{
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
				ContainerImage: ptr.To("http://localhost/nginx:v1.2.3"),
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
