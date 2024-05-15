package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerificationRequestSpec struct {
	User     string `json:"user"`
	Code     string `json:"code"`
	BodyHTML string `json:"bodyHTML"`
	BodyText string `json:"bodyText"`
}

type VerificationRequestStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
type VerificationRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerificationRequestSpec   `json:"spec,omitempty"`
	Status VerificationRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type VerificationRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VerificationRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerificationRequest{}, &VerificationRequestList{})
}
