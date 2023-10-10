package deployment

import (
	"errors"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/go-git/go-git/v5"
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

func TestCreateRepository(t *testing.T) {
	tt := []struct {
		name       string
		deployment v1.Deployment
	}{
		{
			name: "test container image",
			deployment: v1.Deployment{
				Id:             uuid.MustParse("56104a5b-e229-471e-84bd-8217287ab157"),
				Type:           v1.DeploymentTypeContainerImage,
				Name:           util.Ptr("test"),
				Namespace:      util.Ptr("test"),
				ContainerImage: util.Ptr("test"),
			},
		},
		{
			name: "test kustomize",
			deployment: v1.Deployment{
				Id:        uuid.MustParse("bee3f0a1-73e5-45ee-ad06-c535d9c7ac8f"),
				Type:      v1.DeploymentTypeKustomize,
				Name:      util.Ptr("test"),
				Namespace: util.Ptr("test"),
				Kustomize: &map[string][]byte{
					"kustomization.yaml": []byte("test"),
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			dirTemp, err := os.MkdirTemp("", "deployment-")
			if err != nil {
				t.Fatalf("unxepected error creating temporary directory: %s", err)
			}

			err = CreateRepository(&tc.deployment, dirTemp)
			if err != nil {
				t.Fatalf("error creating repository: %s", err)
			}

			gitPath := path.Join(dirTemp, "v1/deployments", tc.deployment.Id.String())

			repository, err := git.PlainOpen(gitPath)
			if err != nil {
				t.Fatalf("error opening git repository: %s", err)
			}

			_, err = repository.ResolveRevision("refs/heads/main")
			if err != nil {
				t.Fatalf("error resolving revision 'refs/heads/main'")
			}
		})
	}
}

func TestCreateRepositoryErrors(t *testing.T) {
	tt := []struct {
		name       string
		deployment v1.Deployment
		expected   error
	}{
		{
			name:       "test empty deployment",
			deployment: v1.Deployment{},
			expected:   ErrUnknownDeploymentType,
		},
		{
			name: "test empty type",
			deployment: v1.Deployment{
				Name: util.Ptr("test"),
			},
			expected: ErrUnknownDeploymentType,
		},
		{
			name: "test empty name",
			deployment: v1.Deployment{
				Type: v1.DeploymentTypeContainerImage,
			},
			expected: ErrDeploymentNameEmpty,
		},
		{
			name: "test helm type",
			deployment: v1.Deployment{
				Name: util.Ptr("test"),
				Type: v1.DeploymentTypeHelm,
			},
			expected: ErrUnknownDeploymentType,
		},
		{
			name: "test empty container image",
			deployment: v1.Deployment{
				Name: util.Ptr("test"),
				Type: v1.DeploymentTypeContainerImage,
			},
			expected: ErrDeploymentImageEmpty,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := CreateRepository(&tc.deployment, "/")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !errors.Is(err, tc.expected) {
				t.Errorf("expected error '%s', got '%s'", tc.expected, err)
			}
		})
	}
}
