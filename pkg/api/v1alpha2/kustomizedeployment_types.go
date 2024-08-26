package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KustomizeDeploymentKind = "KustomizeDeployment"
)

type KustomizeDeploymentSpec struct {
	Kustomize map[string][]byte `json:"kustomize"`
}

type KustomizeDeploymentStatus struct {
	RepositoryURL string             `json:"repositoryURL,omitempty"`
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type KustomizeDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KustomizeDeploymentSpec   `json:"spec,omitempty"`
	Status KustomizeDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KustomizeDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KustomizeDeployment `json:"items"`
}

func (d *KustomizeDeployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *KustomizeDeployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&KustomizeDeployment{}, &KustomizeDeploymentList{})
}
