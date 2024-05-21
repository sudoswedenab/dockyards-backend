package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerificationRequestSpec struct {
	// Deprecated: use UserRef
	User     string                           `json:"user"`
	Code     string                           `json:"code"`
	Subject  string                           `json:"subject"`
	BodyHTML string                           `json:"bodyHTML"`
	BodyText string                           `json:"bodyText"`
	UserRef  corev1.TypedLocalObjectReference `json:"userRef"`
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

func (r *VerificationRequest) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *VerificationRequest) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&VerificationRequest{}, &VerificationRequestList{})
}
