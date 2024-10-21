package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FeatureSpec struct{}

type FeatureStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:unservedversion
type Feature struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FeatureSpec   `json:"spec,omitempty"`
	Status FeatureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:unservedversion
type FeatureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Feature `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Feature{}, &FeatureList{})
}
