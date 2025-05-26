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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodePoolKind = "NodePool"
)

const StorageResourceTypeHostPath = "HostPath"

type NodePoolStorageResource struct {
	Name     string            `json:"name"`
	Type     string            `json:"type,omitempty"`
	Quantity resource.Quantity `json:"quantity"`
}

type NodePoolSecurity struct {
	EnableAppArmor bool `json:"enableAppArmor,omitempty"`
}

type NodePoolSpec struct {
	Replicas         *int32                       `json:"replicas,omitempty"`
	ControlPlane     bool                         `json:"controlPlane,omitempty"`
	LoadBalancer     bool                         `json:"loadBalancer,omitempty"`
	DedicatedRole    bool                         `json:"dedicatedRole,omitempty"`
	Resources        corev1.ResourceList          `json:"resources,omitempty"`
	Storage          bool                         `json:"storage,omitempty"`
	StorageResources []NodePoolStorageResource    `json:"storageResources,omitempty"`
	ReleaseRef       *corev1.TypedObjectReference `json:"releaseRef,omitempty"`
	Security         NodePoolSecurity             `json:"security,omitempty"`
}

type NodePoolStatus struct {
	Conditions       []metav1.Condition  `json:"conditions,omitempty"`
	ClusterServiceID string              `json:"clusterServiceID,omitempty"`
	Resources        corev1.ResourceList `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NodePool `json:"items,omitempty"`
}

func (p *NodePool) GetConditions() []metav1.Condition {
	return p.Status.Conditions
}

func (p *NodePool) SetConditions(conditions []metav1.Condition) {
	p.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&NodePool{}, &NodePoolList{})
}
