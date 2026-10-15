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
	VerificationRequestKind = "VerificationRequest"
	VerifiedCondition       = "Verified"
)

// VerificationRequestSpec defines the desired verification request payload.
type VerificationRequestSpec struct {
	// Code is the verification code expected from the user.
	Code     string                           `json:"code"`
	// Subject is the message subject line sent to the user.
	Subject  string                           `json:"subject"`
	// BodyHTML is the optional HTML message body.
	BodyHTML string                           `json:"bodyHTML,omitempty"`
	// BodyText is the plain-text message body.
	BodyText string                           `json:"bodyText"`
	// UserRef references the user receiving this verification request.
	UserRef  corev1.TypedLocalObjectReference `json:"userRef"`
	// Duration defines how long the verification request remains valid.
	Duration *metav1.Duration                 `json:"duration,omitempty"`
}

// VerificationRequestStatus defines the observed state of a verification request.
type VerificationRequestStatus struct {
	// ProviderID identifies the backing provider verification resource.
	ProviderID string `json:"providerID,omitempty"`
	// ExpirationTimestamp is when this verification request expires.
	ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`
	// Conditions represents the latest available observations of verification state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// VerificationRequest represents a request to verify user identity or ownership.
type VerificationRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired verification request payload.
	Spec   VerificationRequestSpec   `json:"spec,omitempty"`
	// Status represents the observed verification request state.
	Status VerificationRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// VerificationRequestList contains a list of VerificationRequest resources.
type VerificationRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of VerificationRequest resources.
	Items []VerificationRequest `json:"items"`
}

func (r *VerificationRequest) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *VerificationRequest) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func (i *VerificationRequest) GetExpiration() *metav1.Time {
	if i.Spec.Duration == nil {
		return nil
	}

	expiration := i.CreationTimestamp.Add(i.Spec.Duration.Duration)

	return &metav1.Time{Time: expiration}
}

func init() {
	SchemeBuilder.Register(&VerificationRequest{}, &VerificationRequestList{})
}
