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
	KustomizeDeploymentKind = "KustomizeDeployment"
)

type KustomizeDeploymentSpec struct {
	Kustomize map[string][]byte `json:"kustomize"`
}

type KustomizeDeploymentStatus struct {
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
	RepositoryURL string             `json:"repositoryURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:deprecatedversion
type KustomizeDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KustomizeDeploymentSpec   `json:"spec,omitempty"`
	Status KustomizeDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KustomizeDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KustomizeDeployment `json:"items"`
}

func (d *KustomizeDeployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *KustomizeDeployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&KustomizeDeployment{}, &KustomizeDeploymentList{})
}
