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

// NodeClassOpenStackFlavorSpec defines the OpenStack flavor identity used for machine provisioning.
// +kubebuilder:validation:XValidation:rule="has(self.id) || has(self.name)",message="at least one of id or name must be set"
type NodeClassOpenStackFlavorSpec struct {
	// ID is the OpenStack flavor ID and takes precedence over Name when both are set.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id,omitempty"`
	// Name is the OpenStack flavor name used as a fallback when ID is not set.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
}

// NodeClassOpenStackSpec defines OpenStack provider-specific machine settings for this node class.
type NodeClassOpenStackSpec struct {
	// Flavor defines the OpenStack flavor used for machines referencing this node class.
	Flavor NodeClassOpenStackFlavorSpec `json:"flavor"`
}

// NodeClassOpenStackFlavorStatus describes observed OpenStack flavor details.
type NodeClassOpenStackFlavorStatus struct {
	// ID is the observed OpenStack flavor ID.
	ID string `json:"id,omitempty"`
	// Name is the observed OpenStack flavor name.
	Name string `json:"name,omitempty"`
	// VCPUs is the observed number of virtual CPUs in the flavor.
	VCPUs int32 `json:"vcpus,omitempty"`
	// RAMMiB is the observed memory size in mebibytes.
	RAMMiB int32 `json:"ramMiB,omitempty"`
	// DiskGiB is the observed root disk size in gibibytes.
	DiskGiB int32 `json:"diskGiB,omitempty"`
	// EphemeralGiB is the observed ephemeral disk size in gibibytes.
	EphemeralGiB int32 `json:"ephemeralGiB,omitempty"`
	// SwapMiB is the observed swap size in mebibytes.
	SwapMiB int32 `json:"swapMiB,omitempty"`
	// IsPublic indicates whether the observed flavor is public in OpenStack.
	IsPublic bool `json:"isPublic,omitempty"`
}

// NodeClassOpenStackStatus defines observed OpenStack provider-specific machine settings.
type NodeClassOpenStackStatus struct {
	// Flavor describes the observed OpenStack flavor details.
	Flavor *NodeClassOpenStackFlavorStatus `json:"flavor,omitempty"`
}

// NodeClassStatus defines the observed state of a node class.
type NodeClassStatus struct {
	// OpenStack contains observed OpenStack provider-specific machine settings.
	OpenStack *NodeClassOpenStackStatus `json:"openstack,omitempty"`
}

// NodeClassSpec defines reusable scheduling rules for selecting nodes.
type NodeClassSpec struct {
	// NodeSelector matches nodes by exact label key and value.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// NodeAffinity defines additional node selection constraints.
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	// Tolerations allow scheduling onto nodes with matching taints.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:validation:Enum=Dockyards;User
	// Provenience indicates who created this node class definition.
	Provenience string `json:"provenience,omitempty"`

	// OpenStack defines OpenStack provider-specific machine settings for this node class.
	OpenStack *NodeClassOpenStackSpec `json:"openstack,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provenience",type=string,JSONPath=".spec.provenience"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// NodeClass defines reusable node scheduling constraints that can be referenced by workloads.
type NodeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines reusable node scheduling constraints.
	Spec NodeClassSpec `json:"spec,omitempty"`
	// Status represents the observed node class state.
	Status NodeClassStatus `json:"status,omitempty"`
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
