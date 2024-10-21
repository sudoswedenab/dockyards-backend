package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Deprecated: superseded by v1alpha2.ClusterKind
	ClusterKind = "Cluster"
)

type ClusterSpec struct {
	Version                  string `json:"version,omitempty"`
	NoDefaultIngressProvider bool   `json:"noDefaultIngressProvider,omitempty"`
}

type ClusterStatus struct {
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
	ClusterServiceID string             `json:"clusterServiceID,omitempty"`
	Version          string             `json:"version,omitempty"`
	DNSZones         []string           `json:"dnsZones,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=".status.version"
// Deprecated: superseded by v1alpha2.Cluster
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// Deprecated: superseded by v1alpha2.ClusterList
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items,omitempty"`
}

func (c *Cluster) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *Cluster) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
