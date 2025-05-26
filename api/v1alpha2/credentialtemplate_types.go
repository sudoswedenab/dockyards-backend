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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CredentialTemplateKind = "CredentialTemplate"
)

type CredentialOption struct {
	Default     string `json:"default,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Key         string `json:"key"`
	Plaintext   bool   `json:"plaintext,omitempty"`
	Type        string `json:"type,omitempty"`
}

type CredentialTemplateSpec struct {
	Options []CredentialOption `json:"options"`
}

// +kubebuilder:object:root=true
// +kubebuilder:deprecatedversion
type CredentialTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CredentialTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type CredentialTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CredentialTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CredentialTemplate{}, &CredentialTemplateList{})
}
