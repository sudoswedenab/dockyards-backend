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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MemberKind = "Member"
)

const (
	RoleSuperUser = "SuperUser"
	RoleUser      = "User"
	RoleReader    = "Reader"
)

// +kubebuilder:validation:Enum=SuperUser;User;Reader
type Role string

type MemberSpec struct {
	Role    Role                             `json:"role"`
	UserRef corev1.TypedLocalObjectReference `json:"userRef"`
}

type MemberStatus struct {
	Conditions  []metav1.Condition `json:"conditions,omitempty"`
	DisplayName *string            `json:"displayName,omitempty"`
	Email       *string            `json:"email,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=".spec.role"
// +kubebuilder:printcolumn:name="UserName",type=string,JSONPath=".spec.userRef.name"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type Member struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemberSpec   `json:"spec,omitempty"`
	Status MemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type MemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Member `json:"items,omitempty"`
}

func (m *Member) GetConditions() []metav1.Condition {
	return m.Status.Conditions
}

func (m *Member) SetConditions(conditions []metav1.Condition) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Member{}, &MemberList{})
}
