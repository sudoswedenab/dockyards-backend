package v1alpha3

import (
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterKind                         = "Cluster"
	ResourceCluster corev1.ResourceName = "cluster"
)

type ClusterUpgrade struct {
	To string `json:"to"`
}

type ClusterAPIEndpoint struct {
	Host string `json:"host"`
	Port int32  `json:"port"`
}

func (e *ClusterAPIEndpoint) IsValid() bool {
	return e.Host != "" && e.Port != 0
}

func (e *ClusterAPIEndpoint) String() string {
	port := fmt.Sprintf("%d", e.Port)

	return "https://" + net.JoinHostPort(e.Host, port)
}

type ClusterSpec struct {
	Version                  string                            `json:"version,omitempty"`
	NoDefaultIngressProvider bool                              `json:"noDefaultIngressProvider,omitempty"`
	Upgrades                 []ClusterUpgrade                  `json:"upgrades,omitempty"`
	BlockDeletion            bool                              `json:"blockDeletion,omitempty"`
	AllocateInternalIP       bool                              `json:"allocateInternalIP,omitempty"`
	IPPoolRef                *corev1.TypedLocalObjectReference `json:"ipPoolRef,omitempty"`
	Duration                 *metav1.Duration                  `json:"duration,omitempty"`
}

type ClusterStatus struct {
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	ClusterServiceID    string             `json:"clusterServiceID,omitempty"`
	Version             string             `json:"version,omitempty"`
	DNSZones            []string           `json:"dnsZones,omitempty"`
	APIEndpoint         ClusterAPIEndpoint `json:"apiEndpoint,omitempty"`
	ExpirationTimestamp *metav1.Time       `json:"expirationTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,priority=1,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=".status.version"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=".spec.duration"
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
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

func (c *Cluster) GetExpiration() *metav1.Time {
	if c.Spec.Duration == nil {
		return nil
	}

	expiration := c.CreationTimestamp.Add(c.Spec.Duration.Duration)

	return &metav1.Time{Time: expiration}
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
