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
	CredentialTemplateKind = "CredentialTemplate"
)

// CredentialOption defines one configurable credential input.
type CredentialOption struct {
	// Default is the prefilled value for this option.
	Default     string `json:"default,omitempty"`
	// DisplayName is the human-readable label shown for this option.
	DisplayName string `json:"displayName,omitempty"`
	// Key is the unique identifier used when storing this option value.
	Key         string `json:"key"`
	// Plaintext indicates whether the value should be treated as non-secret.
	Plaintext   bool   `json:"plaintext,omitempty"`
	// Type defines the expected input type for this option.
	Type        string `json:"type,omitempty"`
}

// CredentialTemplateSpec defines a reusable set of credential options.
type CredentialTemplateSpec struct {
	// Options contains the credential fields required by this template.
	Options []CredentialOption `json:"options"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// CredentialTemplate defines reusable credential input schema.
type CredentialTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the credential template options.
	Spec CredentialTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// CredentialTemplateList contains a list of CredentialTemplate resources.
type CredentialTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of CredentialTemplate resources.
	Items []CredentialTemplate `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CredentialTemplate{}, &CredentialTemplateList{})
}
