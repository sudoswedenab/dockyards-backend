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

// WorkloadSpec defines the desired workload deployment configuration.
type WorkloadSpec struct {
	// ClusterComponent marks this workload as a cluster-level component.
	ClusterComponent bool   `json:"clusterComponent,omitempty"`
	// TargetNamespace is the namespace where workload resources are applied.
	TargetNamespace  string `json:"targetNamespace"`

	// Deprecated: Use input instead.
	// WorkloadTemplateInput contains legacy template input values.
	WorkloadTemplateInput *apiextensionsv1.JSON `json:"workloadTemplateInput,omitempty"`

	// Input contains template input values for rendering this workload.
	Input               *apiextensionsv1.JSON        `json:"input,omitempty"`
	// WorkloadTemplateRef references the workload template to render.
	WorkloadTemplateRef *corev1.TypedObjectReference `json:"workloadTemplateRef,omitempty"`

	// +kubebuilder:validation:Enum=Dockyards;User
	// Provenience indicates who created this workload definition.
	Provenience string `json:"provenience"`
}

// Deprecated: Superseded by workload inventories type. Will be removed in the next version.
// WorkloadReference defines a legacy reference to a related workload.
type WorkloadReference struct {
	// TypedObjectReference identifies the referenced workload object.
	corev1.TypedObjectReference `json:",inline"`
	// Parent references a parent workload in the legacy dependency graph.
	Parent                      *corev1.TypedLocalObjectReference `json:"parent,omitempty"`
	// URLs lists URLs associated with the referenced workload.
	URLs                        []string                          `json:"urls,omitempty"`
}

// WorkloadStatus defines the observed state of a workload.
type WorkloadStatus struct {
	// Conditions represents the latest available observations of workload state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Deprecated: Superseded by workload inventories type. Will be removed in the next version.
	// DependencyRefs contains legacy workload dependency references.
	DependencyRefs []corev1.TypedLocalObjectReference `json:"dependencyRefs,omitempty"`
	// URLs contains currently observed URLs exposed by this workload.
	URLs           []string                           `json:"urls,omitempty"`

	// Deprecated: Superseded by workload inventories type. Will be removed in the next version.
	// References contains legacy workload reference metadata.
	References []WorkloadReference `json:"references,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="WorkloadTemplate",type=string,priority=1,JSONPath=".spec.workloadTemplateRef.name"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// Workload represents an instantiated deployment based on a workload template.
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired workload deployment configuration.
	Spec   WorkloadSpec   `json:"spec,omitempty"`
	// Status represents the observed workload state.
	Status WorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// WorkloadList contains a list of Workload resources.
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of Workload resources.
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
