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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WorkloadKind = "Workload"
)

type WorkloadSpec struct {
	ClusterComponent bool   `json:"clusterComponent,omitempty"`
	TargetNamespace  string `json:"targetNamespace"`

	// Deprecated: Use input instead.
	WorkloadTemplateInput *apiextensionsv1.JSON `json:"workloadTemplateInput,omitempty"`

	Input               *apiextensionsv1.JSON        `json:"input,omitempty"`
	WorkloadTemplateRef *corev1.TypedObjectReference `json:"workloadTemplateRef,omitempty"`

	// +kubebuilder:validation:Enum=Dockyards;User
	Provenience string `json:"provenience"`
}

type WorkloadReference struct {
	corev1.TypedObjectReference `json:",inline"`
	Parent                      *corev1.TypedLocalObjectReference `json:"parent,omitempty"`
	URLs                        []string                          `json:"urls,omitempty"`
}

type WorkloadStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Deprecated: Use references instead.
	DependencyRefs []corev1.TypedLocalObjectReference `json:"dependencyRefs,omitempty"`
	URLs           []string                           `json:"urls,omitempty"`
	References     []WorkloadReference                `json:"references,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,priority=1,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadSpec   `json:"spec,omitempty"`
	Status WorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workload `json:"items,omitempty"`
}

func (w *Workload) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *Workload) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Workload{}, &WorkloadList{})
}
