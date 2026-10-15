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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WorkloadInventoryKind = "WorkloadInventory"
)

// WorkloadInventorySpec defines how workloads are selected and exposed.
type WorkloadInventorySpec struct {
	// Selector matches workloads included in this inventory.
	Selector metav1.LabelSelector `json:"selector"`

	// URLs lists externally reachable URLs aggregated for matching workloads.
	URLs []string `json:"urls,omitempty"`
}

// +kubebuilder:object:root=true
// WorkloadInventory represents an inventory of workloads selected by labels.
type WorkloadInventory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the workload inventory selection and output settings.
	Spec WorkloadInventorySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// WorkloadInventoryList contains a list of WorkloadInventory resources.
type WorkloadInventoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of WorkloadInventory resources.
	Items []WorkloadInventory `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&WorkloadInventory{}, &WorkloadInventoryList{})
}
