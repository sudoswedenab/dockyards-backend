package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UserKind = "User"
)

type UserSpec struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Password    string `json:"password"`
}

type UserStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Verified",type=string,JSONPath=".status.conditions[?(@.type==\"Verified\")].status"
// +kubebuilder:printcolumn:name="UID",type=string,JSONPath=".metadata.uid"
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
