package v1alpha3

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentKind = "Deployment"
)

type DeploymentSpec struct {
	TargetNamespace         string                             `json:"targetNamespace,omitempty"`
	DeploymentRefs          []corev1.TypedLocalObjectReference `json:"deploymentRefs,omitempty"`
	ClusterComponent        bool                               `json:"clusterComponent,omitempty"`
	DeploymentTemplateRef   *corev1.TypedObjectReference       `json:"deploymentTemplateRef,omitempty"`
	DeploymentTemplateInput *apiextensionsv1.JSON              `json:"deploymentTemplateInput,omitempty"`

	// +kubebuilder:validation:Enum=Dockyards;User
	Provenience string `json:"provenience"`
}

type DeploymentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	URLs       []string           `json:"urls,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:deprecatedversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Provenience",type=string,JSONPath=".spec.provenience"
// +kubebuilder:printcolumn:name="ClusterComponent",type=boolean,priority=1,JSONPath=".spec.clusterComponent"
// +kubebuilder:printcolumn:name="URL",type=string,priority=1,JSONPath=".status.urls[0]"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Deployment `json:"items"`
}

func (d *Deployment) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *Deployment) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
