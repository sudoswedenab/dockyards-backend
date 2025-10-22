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

const (
	OrganizationKind = "Organization"
)

type Cloud struct {
	ProjectRef *NamespacedObjectReference `json:"cloudRef,omitempty"`
	SecretRef  *NamespacedSecretReference `json:"cloudSecret,omitempty"`
}

type OrganizationSpec struct {
	DisplayName string `json:"displayName,omitempty"`

	MemberRefs []MemberReference `json:"memberRefs"`

	BillingRef *NamespacedObjectReference `json:"billingRef,omitempty"`
	Cloud      Cloud                      `json:"cloud,omitempty"`

	SkipAutoAssign bool             `json:"skipAutoAssign,omitempty"`
	Duration       *metav1.Duration `json:"duration,omitempty"`
}

type OrganizationStatus struct {
	Conditions          []metav1.Condition  `json:"conditions,omitempty"`
	NamespaceRef        string              `json:"namespaceRef,omitempty"`
	ResourceQuotas      corev1.ResourceList `json:"resourceQuotas,omitempty"`
	ExpirationTimestamp *metav1.Time        `json:"expirationTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,priority=1,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="NamespaceReference",type=string,JSONPath=".status.namespaceRef"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,JSONPath=".spec.duration"
// +kubebuilder:unservedversion
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Organization `json:"items"`
}

func (o *Organization) GetConditions() []metav1.Condition {
	return o.Status.Conditions
}

func (o *Organization) SetConditions(conditions []metav1.Condition) {
	o.Status.Conditions = conditions
}

func (o *Organization) GetExpiration() *metav1.Time {
	if o.Spec.Duration == nil {
		return nil
	}

	expiration := o.CreationTimestamp.Add(o.Spec.Duration.Duration)

	return &metav1.Time{Time: expiration}
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
