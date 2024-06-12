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

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type HelmDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HelmDeploymentSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type HelmDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []HelmDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmDeployment{}, &HelmDeploymentList{})
}
