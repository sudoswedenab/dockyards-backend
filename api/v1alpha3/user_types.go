// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UserKind = "User"
)

// UserSpec defines the desired user account configuration.
type UserSpec struct {
	// Email is the user's primary email address.
	Email       string           `json:"email"`
	// DisplayName is the human-readable name for the user.
	DisplayName string           `json:"displayName,omitempty"`
	// Password is the initial password value for this user.
	Password    string           `json:"password"`
	// Phone is the user's phone number.
	Phone       string           `json:"phone,omitempty"`
	// Duration defines how long the user account remains active from creation time.
	Duration    *metav1.Duration `json:"duration,omitempty"`
	// ProviderID identifies the backing provider user resource.
	ProviderID  string           `json:"providerID,omitempty"`
}

// UserStatus defines the observed state of a user account.
type UserStatus struct {
	// Conditions represents the latest available observations of user state.
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	// ExpirationTimestamp is when the user account is expected to expire.
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
// User represents a cluster-scoped user account managed by Dockyards.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired user account configuration.
	Spec   UserSpec   `json:"spec,omitempty"`
	// Status represents the observed user account state.
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// UserList contains a list of User resources.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of User resources.
	Items []User `json:"items"`
}

func (u *User) GetConditions() []metav1.Condition {
	return u.Status.Conditions
}

func (u *User) SetConditions(conditions []metav1.Condition) {
	u.Status.Conditions = conditions
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
