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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentTemplateKind = "DeploymentTemplate"
)

type DeploymentTemplateSpec struct {
	CUE string `json:"cue"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type DeploymentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeploymentTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type DeploymentTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DeploymentTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&DeploymentTemplate{}, &DeploymentTemplateList{})
}
