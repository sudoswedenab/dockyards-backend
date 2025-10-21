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
	DNSZoneClaimKind = "DNSZoneClaim"
)

type DNSZoneClaimStatus struct {
	Conditions []metav1.Condition                `json:"conditions,omitempty"`
	DNSZoneRef *corev1.TypedLocalObjectReference `json:"dnsZoneRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
type DNSZoneClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,inline"`

	Status DNSZoneClaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DNSZoneClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DNSZoneClaim `json:"items,omitempty"`
}

func (c *DNSZoneClaim) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *DNSZoneClaim) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&DNSZoneClaim{}, &DNSZoneClaimList{})
}
