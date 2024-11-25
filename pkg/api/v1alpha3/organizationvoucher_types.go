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
	OrganizationVoucherKind = "OrganizationVoucher"
)

type OrganizationVoucherSpec struct {
	Code    string                       `json:"code"`
	PoolRef *corev1.TypedObjectReference `json:"poolRef"`
}

type OrganizationVoucherStatus struct {
	Redeemed bool `json:"redeemed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="PoolReference",type=string,JSONPath=".spec.poolRef.name"
// +kubebuilder:printcolumn:name="Code",type=string,JSONPath=".spec.code"
// +kubebuilder:printcolumn:name="Redeemed",type=boolean,JSONPath=".status.redeemed"
type OrganizationVoucher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationVoucherSpec   `json:"spec,omitempty"`
	Status OrganizationVoucherStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type OrganizationVoucherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OrganizationVoucher `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OrganizationVoucher{}, &OrganizationVoucherList{})
}
