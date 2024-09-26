package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentTemplateKind = "DeploymentTemplate"
)

type DeploymentTemplateSpec struct {
	CUE string `json:"cue"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type DeploymentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeploymentTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type DeploymentTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DeploymentTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&DeploymentTemplate{}, &DeploymentTemplateList{})
}
