package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentKind = "Deployment"
)

type DeploymentSpec struct {
	TargetNamespace string              `json:"targetNamespace,omitempty"`
	DeploymentRef   DeploymentReference `json:"deploymentRef,omitempty"`
}

type DeploymentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	URLs       []string           `json:"urls,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=".status.urls[0]"
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Deployment `json:"items"`
}

func (d *Deployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *Deployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
