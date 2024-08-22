package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
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

	MemberRefs []MemberReference `json:"memberRefs"`

	BillingRef *NamespacedObjectReference `json:"billingRef,omitempty"`
	Cloud      Cloud                      `json:"cloud,omitempty"`

	SkipAutoAssign bool `json:"skipAutoAssign,omitempty"`
}

type OrganizationStatus struct {
	Conditions     []metav1.Condition  `json:"conditions,omitempty"`
	NamespaceRef   string              `json:"namespaceRef,omitempty"`
	ResourceQuotas corev1.ResourceList `json:"resourceQuotas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="NamespaceReference",type=string,JSONPath=".status.namespaceRef"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
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
