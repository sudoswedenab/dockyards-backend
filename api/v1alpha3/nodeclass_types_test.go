// Copyright 2026 Sudo Sweden AB
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

package v1alpha3_test

import (
	"encoding/json"
	"testing"

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNodeClassStatusResourcesRoundTrip(t *testing.T) {
	original := dockyardsv1.NodeClass{
		Status: dockyardsv1.NodeClassStatus{
			Resources: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("8"),
				corev1.ResourceMemory: resource.MustParse("15Gi"),
			},
			OpenStack: &dockyardsv1.NodeClassOpenStackStatus{
				Flavor: &dockyardsv1.NodeClassOpenStackFlavorStatus{RAMMiB: 15360},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded dockyardsv1.NodeClass
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for name, expected := range original.Status.Resources {
		actual, found := decoded.Status.Resources[name]
		if !found || actual.Cmp(expected) != 0 {
			t.Errorf("resource %s: expected %s, got %s (found: %t)", name, expected.String(), actual.String(), found)
		}
	}
	if decoded.Status.OpenStack.Flavor.RAMMiB != 15360 {
		t.Fatal("provider-specific status was not preserved")
	}
}

func TestNodeClassStatusResourcesDeepCopy(t *testing.T) {
	original := &dockyardsv1.NodeClass{
		Status: dockyardsv1.NodeClassStatus{
			Resources: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("15Gi"),
			},
		},
	}
	copied := original.DeepCopy()
	copied.Status.Resources[corev1.ResourceMemory] = resource.MustParse("32Gi")
	if original.Status.Resources.Memory().Cmp(resource.MustParse("15Gi")) != 0 {
		t.Fatal("deep copy shares the resources map")
	}

	empty := (&dockyardsv1.NodeClass{}).DeepCopy()
	if empty.Status.Resources != nil {
		t.Fatal("missing resources must remain absent")
	}
}
