package v1alpha3

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WorkloadKind = "Workload"
)

type WorkloadSpec struct {
	ClusterComponent      bool                         `json:"clusterComponent,omitempty"`
	TargetNamespace       string                       `json:"targetNamespace"`
	WorkloadTemplateInput *apiextensionsv1.JSON        `json:"workloadTemplateInput,omitempty"`
	WorkloadTemplateRef   *corev1.TypedObjectReference `json:"workloadTemplateRef,omitempty"`

	// +kubebuilder:validation:Enum=Dockyards;User
	Provenience string `json:"provenience"`
}

type WorkloadStatus struct {
	Conditions     []metav1.Condition                 `json:"conditions,omitempty"`
	DependencyRefs []corev1.TypedLocalObjectReference `json:"dependencyRefs,omitempty"`
	URLs           []string                           `json:"urls,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadSpec   `json:"spec,omitempty"`
	Status WorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workload `json:"items,omitempty"`
}

func (w *Workload) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *Workload) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Workload{}, &WorkloadList{})
}
