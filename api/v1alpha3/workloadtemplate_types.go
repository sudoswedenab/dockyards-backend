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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkloadTemplateType string

const (
	WorkloadTemplateTypeCue WorkloadTemplateType = "dockyards.io/cue"
)

const (
	WorkloadTemplateKind = "WorkloadTemplate"
)

// WorkloadTemplateSpec defines the desired workload template content.
type WorkloadTemplateSpec struct {
	// Source contains the template source content.
	Source string               `json:"source,omitempty"`
	// Type identifies the template format used in Source.
	Type   WorkloadTemplateType `json:"type"`
}

// WorkloadTemplateStatus defines the observed state of a workload template.
type WorkloadTemplateStatus struct {
	// InputSchema describes accepted input values for this template.
	InputSchema *apiextensionsv1.JSON `json:"inputSchema,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// WorkloadTemplate represents a reusable workload definition template.
type WorkloadTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired workload template content.
	Spec   WorkloadTemplateSpec   `json:"spec,omitempty"`
	// Status represents the observed workload template state.
	Status WorkloadTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// WorkloadTemplateList contains a list of WorkloadTemplate resources.
type WorkloadTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of WorkloadTemplate resources.
	Items []WorkloadTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&WorkloadTemplate{}, &WorkloadTemplateList{})
}
