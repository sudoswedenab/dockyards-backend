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

// IdentityProviderSpec defines the desired identity provider configuration.
type IdentityProviderSpec struct {
	// DisplayName is the human-readable name shown for this identity provider.
	DisplayName   *string                 `json:"displayName,omitempty"`
	// OIDCConfigRef references a secret containing OIDC configuration.
	OIDCConfigRef *corev1.SecretReference `json:"oidc,omitempty"`
}

// OIDCConfig defines OpenID Connect provider and client settings.
type OIDCConfig struct {
	// ClientConfig contains OAuth client credentials and redirect settings.
	ClientConfig         OIDCClientConfig    `json:"clientConfig"`
	// ProviderDiscoveryURL is the OIDC discovery URL used to fetch provider metadata.
	ProviderDiscoveryURL *string             `json:"providerDiscoveryURL,omitempty"`
	// ProviderConfig defines OIDC endpoints explicitly when discovery is not used.
	ProviderConfig       *OIDCProviderConfig `json:"providerConfig,omitempty"`
	// CertificateAuthority is a PEM-encoded CA bundle used to verify provider TLS certificates.
	CertificateAuthority string              `json:"certificateAuthority,omitempty"`
}

// OIDCClientConfig defines OAuth client settings used for OIDC authentication.
type OIDCClientConfig struct {
	// ClientID is the OAuth client identifier.
	ClientID     string `json:"clientID"`
	// RedirectURL is the callback URL registered with the OIDC provider.
	RedirectURL  string `json:"redirectURL"`
	// ClientSecret is the OAuth client secret.
	ClientSecret string `json:"clientSecret,omitempty"`
}

// OIDCProviderConfig defines OIDC provider metadata and endpoint URLs.
type OIDCProviderConfig struct {
	// Issuer is the expected issuer identifier for ID tokens.
	Issuer                      string   `json:"issuer"`
	// AuthorizationEndpoint is the OAuth authorization endpoint URL.
	AuthorizationEndpoint       string   `json:"authorization_endpoint"`
	// TokenEndpoint is the OAuth token endpoint URL.
	TokenEndpoint               string   `json:"token_endpoint"`
	// DeviceAuthorizationEndpoint is the OAuth device authorization endpoint URL.
	DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint,omitempty"`
	// UserinfoEndpoint is the OpenID Connect userinfo endpoint URL.
	UserinfoEndpoint            string   `json:"userinfo_endpoint,omitempty"`
	// JWKSURI is the URL for the provider JSON Web Key Set.
	JWKSURI                     string   `json:"jwks_uri"`
	// IDTokenSigningAlgs lists supported signing algorithms for ID tokens.
	IDTokenSigningAlgs          []string `json:"id_token_signing_alg_values_supported"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// IdentityProvider defines a cluster-scoped identity provider configuration.
type IdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired identity provider configuration.
	Spec IdentityProviderSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// IdentityProviderList contains a list of IdentityProvider resources.
type IdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of IdentityProvider resources.
	Items []IdentityProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdentityProvider{}, &IdentityProviderList{})
}
