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

package v1alpha3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	VirtualMachineInstanceKind                         = "VirtualMachineInstance"
	ResourceVirtualMachineInstance corev1.ResourceName = "virtualmachineinstance"
)

type VirtualMachineInstanceStatus struct {
	Created         bool               `json:"created,omitempty"`
	Ready           bool               `json:"ready,omitempty"`
	PrintableStatus string             `json:"printableStatus,omitempty"`
	Conditions      []metav1.Condition `json:"conditions,omitempty"`
}

type VirtualMachineInstanceSpec struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
type VirtualMachineInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineInstanceSpec   `json:"spec,omitempty"`
	Status VirtualMachineInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type VirtualMachineInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VirtualMachineInstance `json:"items,omitempty"`
}

func (m *VirtualMachineInstance) GetConditions() []metav1.Condition {
	return m.Status.Conditions
}

func (m *VirtualMachineInstance) SetConditions(conditions []metav1.Condition) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&VirtualMachineInstance{}, &VirtualMachineInstanceList{})
}
