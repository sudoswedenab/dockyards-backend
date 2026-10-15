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
	DNSZoneKind = "DNSZone"
)

// DNSZoneSpec defines the desired DNS zone configuration.
type DNSZoneSpec struct {
	// Name is the fully qualified DNS zone name.
	Name       string `json:"name"`
	// ProviderID identifies the external DNS provider zone resource.
	ProviderID string `json:"providerID,omitempty"`
}

// DNSZoneStatus defines the observed state of a DNS zone.
type DNSZoneStatus struct {
	// Conditions represents the latest available observations of DNS zone state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="ZoneName",type=string,JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="ProviderID",type=string,JSONPath=".spec.providerID"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// DNSZone represents a managed DNS zone available to Dockyards resources.
type DNSZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired DNS zone configuration.
	Spec   DNSZoneSpec   `json:"spec,omitempty"`
	// Status represents the observed DNS zone state.
	Status DNSZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// DNSZoneList contains a list of DNSZone resources.
type DNSZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of DNSZone resources.
	Items []DNSZone `json:"items,omitempty"`
}

func (z *DNSZone) GetConditions() []metav1.Condition {
	return z.Status.Conditions
}

func (z *DNSZone) SetConditions(conditions []metav1.Condition) {
	z.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&DNSZone{}, &DNSZoneList{})
}
