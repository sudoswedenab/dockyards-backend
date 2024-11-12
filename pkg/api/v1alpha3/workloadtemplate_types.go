package v1alpha3

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkloadTemplateType string

const (
	WorkloadTemplateTypeCue WorkloadTemplateType = "dockyards.io/cue"
)

const (
	WorkloadTemplateKind = "WorkloadTemplate"
)

type WorkloadTemplateSpec struct {
	Source string               `json:"source,omitempty"`
	Type   WorkloadTemplateType `json:"type"`
}

type WorkloadTemplateStatus struct {
	InputSchema *apiextensionsv1.JSON `json:"inputSchema,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type WorkloadTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadTemplateSpec   `json:"spec,omitempty"`
	Status WorkloadTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type WorkloadTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []WorkloadTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&WorkloadTemplate{}, &WorkloadTemplateList{})
}
