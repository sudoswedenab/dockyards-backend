package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeKind = "Node"
)

type NodeSpec struct {
}

type NodeStatus struct {
	ClusterServiceID string              `json:"clusterServiceID,omitempty"`
	Conditions       []metav1.Condition  `json:"conditions,omitempty"`
	Resources        corev1.ResourceList `json:"resources,omitempty"`
	CloudServiceID   string              `json:"cloudServiceID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
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
