package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OrganizationKind = "Organization"
)

type Cloud struct {
	ProjectRef *NamespacedObjectReference `json:"cloudRef,omitempty"`
	SecretRef  *NamespacedSecretReference `json:"cloudSecret,omitempty"`
}

type OrganizationSpec struct {
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:MinItems=1
	MemberRefs []MemberReference `json:"memberRefs"`

	BillingRef *NamespacedObjectReference `json:"billingRef,omitempty"`
	Cloud      Cloud                      `json:"cloud,omitempty"`
}

type OrganizationStatus struct {
	Conditions   []metav1.Condition `json:"conditions,omitempty"`
	NamespaceRef string             `json:"namespaceRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:storageversion
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Organization `json:"items"`
}

func (o *Organization) GetConditions() []metav1.Condition {
	return o.Status.Conditions
}

func (o *Organization) SetConditions(conditions []metav1.Condition) {
	o.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
