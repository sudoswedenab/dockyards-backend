// Copyright 2025 Sudo Sweden AB
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
	IdentityProviderKind = "IdentityProvider"
)

type IdentityProviderSpec struct {
	DisplayName *string                 `json:"displayName,omitempty"`
	OIDCConfig  *corev1.SecretReference `json:"oidc,omitempty"`
	// OIDCConfig  *OIDCConfig `json:"oidc,omitempty"`
}

type OIDCConfig struct {
	OIDCClientConfig         OIDCClientConfig    `json:"clientConfig"`
	OIDCProviderDiscoveryURL *string             `json:"providerDiscoveryURL,omitempty"`
	OIDCProviderConfig       *OIDCProviderConfig `json:"providerConfig,omitempty"`
}

type OIDCClientConfig struct {
	ClientID     string `json:"clientID"`
	RedirectURL  string `json:"redirectURL"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

// Fields renamed from github.com/coreos/go-oidc ProviderConfig
type OIDCProviderConfig struct {
	Issuer                      string   `json:"issuer"`
	AuthorizationEndpoint       string   `json:"authorizationEndpoint"`
	TokenEndpoint               string   `json:"tokenEndpoint"`
	DeviceAuthorizationEndpoint string   `json:"deviceAuthorizationEndpoint,omitempty"`
	UserinfoEndpoint            string   `json:"userinfoEndpoint,omitempty"`
	JWKSURI                     string   `json:"jwksURI"`
	IDTokenSigningAlgs          []string `json:"idTokenSigningAlgs"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type IdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IdentityProviderSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type IdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []IdentityProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdentityProvider{}, &IdentityProviderList{})
}
