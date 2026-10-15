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
	NodeClassKind = "NodeClass"
)

// NodeClassSpec defines reusable scheduling rules for selecting nodes.
type NodeClassSpec struct {
	// NodeSelector matches nodes by exact label key and value.
	NodeSelector map[string]string    `json:"nodeSelector,omitempty"`
	// NodeAffinity defines additional node selection constraints.
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	// Tolerations allow scheduling onto nodes with matching taints.
	Tolerations  []corev1.Toleration  `json:"tolerations,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// NodeClass defines reusable node scheduling constraints that can be referenced by workloads.
type NodeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines reusable node scheduling constraints.
	Spec NodeClassSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// NodeClassList contains a list of NodeClass resources.
type NodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of NodeClass resources.
	Items []NodeClass `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&NodeClass{}, &NodeClassList{})
}
