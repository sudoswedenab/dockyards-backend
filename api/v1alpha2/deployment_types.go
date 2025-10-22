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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentKind = "Deployment"
)

type DeploymentSpec struct {
	TargetNamespace         string                             `json:"targetNamespace,omitempty"`
	DeploymentRefs          []corev1.TypedLocalObjectReference `json:"deploymentRefs,omitempty"`
	ClusterComponent        bool                               `json:"clusterComponent,omitempty"`
	DeploymentTemplateRef   *corev1.TypedObjectReference       `json:"deploymentTemplateRef,omitempty"`
	DeploymentTemplateInput *apiextensionsv1.JSON              `json:"deploymentTemplateInput,omitempty"`

	// +kubebuilder:validation:Enum=Dockyards;User
	Provenience string `json:"provenience"`
}

type DeploymentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	URLs       []string           `json:"urls,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Provenience",type=string,JSONPath=".spec.provenience"
// +kubebuilder:printcolumn:name="ClusterComponent",type=boolean,priority=1,JSONPath=".spec.clusterComponent"
// +kubebuilder:printcolumn:name="URL",type=string,priority=1,JSONPath=".status.urls[0]"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:unservedversion
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Deployment `json:"items"`
}

func (d *Deployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *Deployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
