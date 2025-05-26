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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeKind = "Node"
)

type NodeSpec struct {
	ProviderID *string `json:"providerID,omitempty"`
}

type NodeStatus struct {
	// Deprecated: This field is deprecated and will be removed in the next version.
	ClusterServiceID string              `json:"clusterServiceID,omitempty"`
	Conditions       []metav1.Condition  `json:"conditions,omitempty"`
	Resources        corev1.ResourceList `json:"resources,omitempty"`

	// Deprecated: This field is deprecated, use spec.providerID instead.
	CloudServiceID string                 `json:"cloudServiceID,omitempty"`
	SystemInfo     *corev1.NodeSystemInfo `json:"systemInfo,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="ProviderID",type=string,JSONPath=".spec.providerID"
// +kubebuilder:printcolumn:name="CPU",type=string,priority=1,JSONPath=".status.resources.cpu"
// +kubebuilder:printcolumn:name="Memory",type=string,priority=1,JSONPath=".status.resources.memory"
// +kubebuilder:printcolumn:name="Storage",type=string,priority=1,JSONPath=".status.resources.storage"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSpec   `json:"spec,omitempty"`
	Status NodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Node `json:"items,omitempty"`
}

func (n *Node) GetConditions() []metav1.Condition {
	return n.Status.Conditions
}

func (n *Node) SetConditions(conditions []metav1.Condition) {
	n.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Node{}, &NodeList{})
}
