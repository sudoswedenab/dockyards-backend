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
	InvitationKind = "Invitation"
)

type InvitationSpec struct {
	Duration  *metav1.Duration             `json:"duration,omitempty"`
	Email     string                       `json:"email"`
	Role      OrganizationMemberRole       `json:"role"`
	SenderRef *corev1.TypedObjectReference `json:"senderRef,omitempty"`
}

// +kubebuilder:printcolumn:name="Email",type=string,JSONPath=".spec.email"
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=".spec.role"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=".spec.duration"
// +kubebuilder:object:root=true
type Invitation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InvitationSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type InvitationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Invitation `json:"items"`
}

func (i *Invitation) GetExpiration() *metav1.Time {
	if i.Spec.Duration == nil {
		return nil
	}

	expiration := i.CreationTimestamp.Add(i.Spec.Duration.Duration)

	return &metav1.Time{Time: expiration}
}

func init() {
	SchemeBuilder.Register(&Invitation{}, &InvitationList{})
}
