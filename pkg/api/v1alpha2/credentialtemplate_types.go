package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CredentialTemplateKind = "CredentialTemplate"
)

type CredentialOption struct {
	Default     string `json:"default,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Key         string `json:"key"`
	Plaintext   bool   `json:"plaintext,omitempty"`
	Type        string `json:"type,omitempty"`
}

type CredentialTemplateSpec struct {
	Options []CredentialOption `json:"options"`
}

// +kubebuilder:object:root=true
type CredentialTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CredentialTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type CredentialTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CredentialTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CredentialTemplate{}, &CredentialTemplateList{})
}
