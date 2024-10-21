package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Deprecated: superseded by v1alpha2.UserKind
	UserKind = "User"
)

type UserSpec struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:Pattern="^\\$2a\\$10\\$[\\.\\/A-Za-z0-9]{53}$"
	Password string `json:"password"`
}

type UserStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Verified",type=string,JSONPath=".status.conditions[?(@.type==\"Verified\")].status"
// +kubebuilder:printcolumn:name="UID",type=string,JSONPath=".metadata.uid"
// +kubebuilder:unservedversion
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:unservedversion
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
