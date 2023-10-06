package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UserKind = "User"
)

type UserSpec struct {
	Email string `json:"email"`

	// +kubebuilder:validation:Pattern="^\\$2a\\$10\\$[\\.\\/A-Za-z0-9]{53}$"
	Password string `json:"password"`
}

type UserStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=.metadata.creationTimestamp
// +kubebuilder:printcolumn:name="Verified",type=string,JSONPath=".status.conditions[?(@.type==\"Verified\")].status"
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
