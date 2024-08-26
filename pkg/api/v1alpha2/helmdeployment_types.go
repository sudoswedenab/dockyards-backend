package v1alpha2

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HelmDeploymentKind = "HelmDeployment"
)

type HelmDeploymentSpec struct {
	Chart         string                `json:"chart"`
	Repository    string                `json:"repository"`
	Version       string                `json:"version"`
	Values        *apiextensionsv1.JSON `json:"values,omitempty"`
	SkipNamespace bool                  `json:"skipNamespace,omitempty"`
}

type HelmDeploymentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type HelmDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmDeploymentSpec   `json:"spec,omitempty"`
	Status HelmDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type HelmDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []HelmDeployment `json:"items"`
}

func (d *HelmDeployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *HelmDeployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&HelmDeployment{}, &HelmDeploymentList{})
}
