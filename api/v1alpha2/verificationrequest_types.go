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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VerificationRequestSpec struct {
	// Deprecated: use UserRef
	User     string                           `json:"user,omitempty"`
	Code     string                           `json:"code"`
	Subject  string                           `json:"subject"`
	BodyHTML string                           `json:"bodyHTML"`
	BodyText string                           `json:"bodyText"`
	UserRef  corev1.TypedLocalObjectReference `json:"userRef"`
}

type VerificationRequestStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:deprecatedversion
type VerificationRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerificationRequestSpec   `json:"spec,omitempty"`
	Status VerificationRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type VerificationRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VerificationRequest `json:"items"`
}

func (r *VerificationRequest) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *VerificationRequest) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&VerificationRequest{}, &VerificationRequestList{})
}
