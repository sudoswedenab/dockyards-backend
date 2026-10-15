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
	ClusterTemplateKind = "ClusterTemplate"
)

// NodePoolTemplate defines a node pool template included in a cluster template.
type NodePoolTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration for this node pool template.
	Spec NodePoolSpec `json:"spec,omitempty"`
}

// ClusterTemplateSpec defines reusable templates for cluster composition.
type ClusterTemplateSpec struct {
	// NodePoolTemplates lists the node pool templates applied to clusters using this template.
	NodePoolTemplates []NodePoolTemplate `json:"nodePoolTemplates,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// ClusterTemplate defines reusable cluster defaults and node pool templates.
type ClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the template configuration used when creating clusters.
	Spec ClusterTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// ClusterTemplateList contains a list of ClusterTemplate resources.
type ClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of ClusterTemplate resources.
	Items []ClusterTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplate{}, &ClusterTemplateList{})
}
