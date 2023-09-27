package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OrganizationKind = "Organization"
)

type OrganizationSpec struct {
	DisplayName string                   `json:"displayName,omitempty"`
	MemberRefs  []corev1.ObjectReference `json:"memberRefs"`
	BillingRef  *corev1.ObjectReference  `json:"billingRef,omitempty"`
	CloudRef    *corev1.ObjectReference  `json:"cloudRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OrganizationSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
