package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Deprecated: superseded by v1alpha2.ClusterTemplateKind
	ClusterTemplateKind = "ClusterTemplate"
)

type ClusterTemplateSpec struct {
	NodePoolTemplates []NodePool `json:"nodePoolTemplates,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// Deprecated: superseded by v1alpha2.ClusterTemplate
type ClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// Deprecated: superseded by v1alpha2.ClusterTemplateList
type ClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ClusterTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ClusterTemplate{}, &ClusterTemplateList{})
}
