package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UserKind = "User"
)

type UserSpec struct {
	Email       string           `json:"email"`
	DisplayName string           `json:"displayName,omitempty"`
	Password    string           `json:"password"`
	Phone       string           `json:"phone,omitempty"`
	Duration    *metav1.Duration `json:"duration,omitempty"`
}

type UserStatus struct {
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	ExpirationTimestamp *metav1.Time       `json:"expirationTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="UID",type=string,priority=1,JSONPath=".metadata.uid"
// +kubebuilder:printcolumn:name="Email",type=string,JSONPath=".spec.email"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=".spec.duration"
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

func (u *User) GetExpiration() *metav1.Time {
	if u.Spec.Duration == nil {
		return nil
	}

	expiration := u.CreationTimestamp.Add(u.Spec.Duration.Duration)

	return &metav1.Time{Time: expiration}
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
