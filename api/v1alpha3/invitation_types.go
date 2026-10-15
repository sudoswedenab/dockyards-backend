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

// InvitationSpec defines the desired invitation details.
type InvitationSpec struct {
	// Duration defines how long the invitation remains valid from creation time.
	Duration  *metav1.Duration             `json:"duration,omitempty"`
	// Email is the recipient email address for the invitation.
	Email     string                       `json:"email"`
	// Role is the organization role granted when the invitation is accepted.
	Role      Role                         `json:"role"`
	// SenderRef references the user or service that created the invitation.
	SenderRef *corev1.TypedObjectReference `json:"senderRef,omitempty"`
}

// InvitationStatus defines the observed state of an invitation.
type InvitationStatus struct {
	// ExpirationTimestamp is when the invitation expires.
	ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`
}

// +kubebuilder:printcolumn:name="Email",type=string,JSONPath=".spec.email"
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=".spec.role"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=".spec.duration"
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Invitation represents an invitation for a user to join an organization.
type Invitation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired invitation details.
	Spec   InvitationSpec   `json:"spec,omitempty"`
	// Status represents the observed invitation state.
	Status InvitationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// InvitationList contains a list of Invitation resources.
type InvitationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of Invitation resources.
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
