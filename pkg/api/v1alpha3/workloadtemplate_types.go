package v1alpha3

import (
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

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type WorkloadTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WorkloadTemplateSpec `json:"spec,omitempty"`
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
