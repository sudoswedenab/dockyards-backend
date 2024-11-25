// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployment

import (
	"testing"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"k8s.io/utils/ptr"
)

func TestAddNormalizedName(t *testing.T) {
	tt := []struct {
		name       string
		deployment types.Deployment
		expected   types.Deployment
	}{
		{
			name: "test container image",
			deployment: types.Deployment{
				ContainerImage: ptr.To("test:types.2.3"),
			},
			expected: types.Deployment{
				Name:           ptr.To("test"),
				Namespace:      ptr.To("test"),
				ContainerImage: ptr.To("docker.io/library/test:types.2.3"),
				Type:           types.DeploymentTypeContainerImage,
			},
		},
		{
			name: "test helm chart",
			deployment: types.Deployment{
				HelmChart: ptr.To("test"),
			},
			expected: types.Deployment{
				Name:      ptr.To("test"),
				Namespace: ptr.To("test"),
				HelmChart: ptr.To("test"),
				Type:      types.DeploymentTypeHelm,
			},
		},
		{
			name: "test kustomize",
			deployment: types.Deployment{
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("test"),
				}),
			},
			expected: types.Deployment{
				Name:      ptr.To(""),
				Namespace: ptr.To(""),
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("test"),
				}),
				Type: types.DeploymentTypeKustomize,
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
		deployment types.Deployment
	}{
		{
			name: "test invalid container image",
			deployment: types.Deployment{
				ContainerImage: ptr.To("http://localhost/nginx:types.2.3"),
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
