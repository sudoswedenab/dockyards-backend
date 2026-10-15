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

// StorageResourceTypeHostPath identifies host-backed storage resources.
const StorageResourceTypeHostPath = "HostPath"

// NodePoolStorageResource defines an additional storage resource exposed by nodes in the pool.
type NodePoolStorageResource struct {
	// Name is the resource name advertised on nodes.
	Name     string            `json:"name"`
	// Type defines the storage backing type.
	Type     string            `json:"type,omitempty"`
	// Quantity is the allocatable amount exposed for this resource.
	Quantity resource.Quantity `json:"quantity"`
}

// NodePoolSecurity defines security-related options for nodes in the pool.
type NodePoolSecurity struct {
	// EnableAppArmor enables AppArmor support on nodes in this pool.
	EnableAppArmor bool `json:"enableAppArmor,omitempty"`
}

// NodePoolSpec defines the desired configuration of a node pool.
type NodePoolSpec struct {
	// Replicas is the desired number of nodes in the pool.
	Replicas         *int32                       `json:"replicas,omitempty"`
	// ControlPlane marks nodes in this pool as control plane nodes.
	ControlPlane     bool                         `json:"controlPlane,omitempty"`
	// LoadBalancer marks nodes in this pool as load balancer nodes.
	LoadBalancer     bool                         `json:"loadBalancer,omitempty"`
	// DedicatedRole prevents scheduling general workloads on these nodes.
	DedicatedRole    bool                         `json:"dedicatedRole,omitempty"`
	// Resources configures requested compute resources per node.
	Resources        corev1.ResourceList          `json:"resources,omitempty"`
	// Storage enables storage capabilities for nodes in this pool.
	Storage          bool                         `json:"storage,omitempty"`
	// StorageResources defines additional storage resource capacities per node.
	StorageResources []NodePoolStorageResource    `json:"storageResources,omitempty"`
	// ReleaseRef references the OS or image release used by this node pool.
	ReleaseRef       *corev1.TypedObjectReference `json:"releaseRef,omitempty"`
	// Security defines security settings applied to nodes in this pool.
	Security         NodePoolSecurity             `json:"security,omitempty"`
	// NodeAnnotations are applied to Kubernetes Node objects created from this pool.
	NodeAnnotations  map[string]string            `json:"nodeAnnotations,omitempty"`
	// NodeLabels are applied to Kubernetes Node objects created from this pool.
	NodeLabels       map[string]string            `json:"nodeLabels,omitempty"`
	// NodeTaints are applied to Kubernetes Node objects created from this pool.
	NodeTaints       map[string]string            `json:"nodeTaints,omitempty"`
	// NodeClassRef references a NodeClass used for node scheduling defaults.
	NodeClassRef     *corev1.TypedObjectReference `json:"nodeClassRef,omitempty"`
}

// NodePoolStatus defines the observed state of a node pool.
type NodePoolStatus struct {
	// Conditions represents the latest available observations of node pool state.
	Conditions       []metav1.Condition  `json:"conditions,omitempty"`
	// ClusterServiceID identifies the corresponding node pool in backend services.
	ClusterServiceID string              `json:"clusterServiceID,omitempty"`
	// Resources reports aggregate observed resources across nodes in this pool.
	Resources        corev1.ResourceList `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// NodePool represents a group of cluster nodes sharing the same configuration.
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired node pool configuration.
	Spec   NodePoolSpec   `json:"spec,omitempty"`
	// Status represents the observed node pool state.
	Status NodePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// NodePoolList contains a list of NodePool resources.
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of NodePool resources.
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
