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
	ctrl "sigs.k8s.io/controller-runtime"
)

// Deprecated: Superseded by role type. Will be removed in the next version.
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
	Role OrganizationMemberRole `json:"role"`
	UID  types.UID              `json:"uid"`
}

type OrganizationSpec struct {
	DisplayName string `json:"displayName,omitempty"`

	// Deprecated: Superseded by the member type. Will be removed in the next version.
	MemberRefs []OrganizationMemberReference `json:"memberRefs,omitempty"`

	ProjectRef    *corev1.TypedObjectReference `json:"projectRef,omitempty"`
	CredentialRef *corev1.TypedObjectReference `json:"credentialRef,omitempty"`

	SkipAutoAssign bool             `json:"skipAutoAssign,omitempty"`
	Duration       *metav1.Duration `json:"duration,omitempty"`

	NamespaceRef *corev1.LocalObjectReference `json:"namespaceRef,omitempty"`
	ProviderID   *string                      `json:"providerID,omitempty"`
}

type OrganizationStatus struct {
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	ExpirationTimestamp *metav1.Time       `json:"expirationTimestamp,omitempty"`

	// Deprecated: use spec.namespaceRef
	NamespaceRef   *corev1.LocalObjectReference `json:"namespaceRef,omitempty"`
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

func (*Organization) Hub() {}

func (o *Organization) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(o).Complete()
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
