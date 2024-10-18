package v1alpha2

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
// +kubebuilder:deprecatedversion
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
