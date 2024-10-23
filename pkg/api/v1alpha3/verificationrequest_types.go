package v1alpha3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerificationRequestSpec struct {
	Code     string                           `json:"code"`
	Subject  string                           `json:"subject"`
	BodyHTML string                           `json:"bodyHTML,omitempty"`
	BodyText string                           `json:"bodyText"`
	UserRef  corev1.TypedLocalObjectReference `json:"userRef"`
}

type VerificationRequestStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
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
