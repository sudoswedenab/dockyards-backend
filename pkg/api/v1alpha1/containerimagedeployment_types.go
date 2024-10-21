package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Deprecated: superseded by v1alpha2.ContainerImageDeploymentKind
	ContainerImageDeploymentKind = "ContainerImageDeployment"
)

type ContainerImageDeploymentSpec struct {
	Image         string                       `json:"image"`
	Port          int32                        `json:"port,omitempty"`
	CredentialRef *corev1.LocalObjectReference `json:"credentialRef,omitempty"`
}

type ContainerImageDeploymentStatus struct {
	RepositoryURL string             `json:"repositoryURL,omitempty"`
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:unservedversion
type ContainerImageDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerImageDeploymentSpec   `json:"spec,omitempty"`
	Status ContainerImageDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:unservedversion
type ContainerImageDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ContainerImageDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerImageDeployment{}, &ContainerImageDeploymentList{})
}
