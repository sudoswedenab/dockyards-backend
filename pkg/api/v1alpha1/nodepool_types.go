package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodePoolKind = "NodePool"
)

type NodePoolSpec struct {
	Replicas      *int32              `json:"replicas,omitempty"`
	ControlPlane  bool                `json:"controlPlane,omitempty"`
	LoadBalancer  bool                `json:"loadBalancer,omitempty"`
	DedicatedRole bool                `json:"dedicatedRole,omitempty"`
	Resources     corev1.ResourceList `json:"resources,omitempty"`
}

type NodePoolStatus struct {
	Conditions       []metav1.Condition  `json:"conditions,omitempty"`
	ClusterServiceID string              `json:"clusterServiceID,omitempty"`
	Resources        corev1.ResourceList `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provisioned",type=string,JSONPath=".status.conditions[?(@.type==\"Provisioned\")].status"
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

func init() {
	SchemeBuilder.Register(&NodePool{}, &NodePoolList{})
}
