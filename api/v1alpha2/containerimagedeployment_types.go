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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ContainerImageDeploymentKind = "ContainerImageDeployment"
)

type ContainerImageDeploymentSpec struct {
	Image         string                       `json:"image"`
	Port          int32                        `json:"port,omitempty"`
	CredentialRef *corev1.LocalObjectReference `json:"credentialRef,omitempty"`
}

type ContainerImageDeploymentStatus struct {
	RepositoryURL string             `json:"repositoryURL,omitempty"`
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:unservedversion
type ContainerImageDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerImageDeploymentSpec   `json:"spec,omitempty"`
	Status ContainerImageDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ContainerImageDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ContainerImageDeployment `json:"items"`
}

func (d *ContainerImageDeployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *ContainerImageDeployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&ContainerImageDeployment{}, &ContainerImageDeploymentList{})
}
