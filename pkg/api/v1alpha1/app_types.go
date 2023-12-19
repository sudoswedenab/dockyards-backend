package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppOption struct {
	JSONPointer string   `json:"jsonPointer"`
	DisplayName string   `json:"displayName"`
	Default     string   `json:"default,omitempty"`
	Type        string   `json:"type,omitempty"`
	Hidden      bool     `json:"hidden,omitempty"`
	Selection   []string `json:"selection,omitempty"`
	Managed     bool     `json:"managed,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Toggle      []string `json:"toggle,omitempty"`
}

type AppStep struct {
	Name    string      `json:"name"`
	Options []AppOption `json:"options,omitempty"`
}

type AppSpec struct {
	Icon        string    `json:"icon,omitempty"`
	Description string    `json:"description,omitempty"`
	Steps       []AppStep `json:"steps"`
}

type AppStatus struct {
	AppID string `json:"appID"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
