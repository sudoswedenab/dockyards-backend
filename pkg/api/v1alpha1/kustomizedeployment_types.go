package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Deprecated: superseded by v1alpha2.KustomizeDeploymentKind
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
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// Deprecated: superseded by v1alpha2.KustomizeDeployment
type KustomizeDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KustomizeDeploymentSpec   `json:"spec,omitempty"`
	Status KustomizeDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// Deprecated: superseded by v1alpha2.KustomizeDeploymentList
type KustomizeDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KustomizeDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KustomizeDeployment{}, &KustomizeDeploymentList{})
}
