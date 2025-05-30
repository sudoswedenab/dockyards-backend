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

type NodePoolTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NodePoolSpec `json:"spec,omitempty"`
}

type ClusterTemplateSpec struct {
	NodePoolTemplates []NodePoolTemplate `json:"nodePoolTemplates,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type ClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type ClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplate{}, &ClusterTemplateList{})
}
