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
	"k8s.io/apimachinery/pkg/types"
)

// Deprecated: Superseded by role type. Will be removed in the next version.
// OrganizationMemberRole defines a legacy role value for organization members.
type OrganizationMemberRole string

const (
	OrganizationKind = "Organization"

	// Deprecated: Superseded by RoleSuperUser constant. Will be removed in the next version.
	OrganizationMemberRoleSuperUser OrganizationMemberRole = "SuperUser"
	// Deprecated: Superseded by RoleUser constant. Will be removed in the next version.
	OrganizationMemberRoleUser      OrganizationMemberRole = "User"
	// Deprecated: Superseded by RoleReader constant. Will be removed in the next version.
	OrganizationMemberRoleReader    OrganizationMemberRole = "Reader"
)

type OrganizationMemberReference struct {
	corev1.TypedLocalObjectReference `json:",inline"`

	// +kubebuilder:validation:Enum=SuperUser;User;Reader
	// Role is the legacy role assigned to the referenced member.
	Role OrganizationMemberRole `json:"role"`
	// UID is the user UID of the referenced member.
	UID  types.UID              `json:"uid"`
}

// OrganizationSpec defines the desired organization configuration.
type OrganizationSpec struct {
	// DisplayName is the human-readable name for the organization.
	DisplayName string `json:"displayName,omitempty"`

	// Deprecated: Superseded by the member type. Will be removed in the next version.
	// MemberRefs contains legacy organization member references.
	MemberRefs []OrganizationMemberReference `json:"memberRefs,omitempty"`

	// ProjectRef references the project associated with this organization.
	ProjectRef    *corev1.TypedObjectReference `json:"projectRef,omitempty"`
	// CredentialRef references credentials used for organization provisioning.
	CredentialRef *corev1.TypedObjectReference `json:"credentialRef,omitempty"`

	// SkipAutoAssign disables automatic assignment behavior for this organization.
	SkipAutoAssign bool             `json:"skipAutoAssign,omitempty"`
	// Duration defines how long the organization remains active from creation time.
	Duration       *metav1.Duration `json:"duration,omitempty"`

	// NamespaceRef references the Kubernetes namespace associated with this organization.
	NamespaceRef *corev1.LocalObjectReference `json:"namespaceRef,omitempty"`
	// ProviderID identifies the backing provider organization resource.
	ProviderID   *string                      `json:"providerID,omitempty"`
}

// OrganizationStatus defines the observed state of an organization.
type OrganizationStatus struct {
	// Conditions represents the latest available observations of organization state.
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	// ExpirationTimestamp is when the organization is expected to expire.
	ExpirationTimestamp *metav1.Time       `json:"expirationTimestamp,omitempty"`

	// Deprecated: use spec.namespaceRef
	// NamespaceRef is the legacy namespace reference for this organization.
	NamespaceRef   *corev1.LocalObjectReference `json:"namespaceRef,omitempty"`
	// ResourceQuotas reports currently observed organization resource quotas.
	ResourceQuotas corev1.ResourceList          `json:"resourceQuotas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="NamespaceReference",type=string,priority=1,JSONPath=".spec.namespaceRef.name"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,priority=1,JSONPath=".spec.duration"
// Organization represents a cluster-scoped tenant organization in Dockyards.
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired organization configuration.
	Spec   OrganizationSpec   `json:"spec"`
	// Status represents the observed organization state.
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// OrganizationList contains a list of Organization resources.
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of Organization resources.
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
