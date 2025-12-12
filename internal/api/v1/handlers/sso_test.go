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

package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

func TestGlobalIdentityProviders_List(t *testing.T) {
	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()
	ns := testEnvironment.GetDockyardsNamespace()

	reqTarget := "/v1/identity-providers"

	t.Run("test no providers", func(t *testing.T) {
		scheme := scheme.Scheme
		dockyardsv1.AddToScheme(scheme)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, reqTarget, nil)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("error reading result body: %s", err)
		}

		var actual []types.IdentityProvider
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling response body: %s", err)
		}

		expected := []types.IdentityProvider{}

		if !cmp.Equal(actual, expected) {
			t.Errorf("difference between actual and expected: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test filtered providers", func(t *testing.T) {
		mustJSON := func(t *testing.T, v any) []byte {
			t.Helper()

			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}

			return b
		}

		toSecretData := func(t *testing.T, cfg dockyardsv1.OIDCConfig) map[string][]byte {
			t.Helper()

			data := map[string][]byte{
				"clientConfig": mustJSON(t, cfg.ClientConfig),
			}

			if cfg.ProviderConfig != nil {
				data["providerConfig"] = mustJSON(t, cfg.ProviderConfig)
			}

			if cfg.ProviderDiscoveryURL != nil {
				data["providerDiscoveryURL"] = []byte(*cfg.ProviderDiscoveryURL)
			}

			return data
		}

		baseClientConfig := dockyardsv1.OIDCClientConfig{
			ClientID:     "client",
			RedirectURL:  "https://redirect.example.com",
			ClientSecret: "secret",
		}

		disc := dockyardsv1.OIDCConfig{
			ClientConfig:         baseClientConfig,
			ProviderDiscoveryURL: ptr.To("https://issuer.example.com"),
		}
		conf := dockyardsv1.OIDCConfig{
			ClientConfig: baseClientConfig,
			ProviderConfig: &dockyardsv1.OIDCProviderConfig{
				Issuer:                "https://issuer.example.com",
				AuthorizationEndpoint: "https://issuer.example.com/auth",
				TokenEndpoint:         "https://issuer.example.com/token",
				JWKSURI:               "https://issuer.example.com/jwks",
				IDTokenSigningAlgs: []string{
					"RS256",
				},
			},
		}
		both := dockyardsv1.OIDCConfig{
			ClientConfig: baseClientConfig,
			ProviderConfig: &dockyardsv1.OIDCProviderConfig{
				Issuer:                "https://issuer.example.com",
				AuthorizationEndpoint: "https://issuer.example.com/auth",
				TokenEndpoint:         "https://issuer.example.com/token",
				JWKSURI:               "https://issuer.example.com/jwks",
				IDTokenSigningAlgs: []string{
					"RS256",
				},
			},
			ProviderDiscoveryURL: ptr.To("https://issuer.example.com"),
		}
		neither := dockyardsv1.OIDCConfig{
			ClientConfig: baseClientConfig,
		}

		idps := []dockyardsv1.IdentityProvider{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-discovery",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfigRef: &corev1.SecretReference{
						Name:      "oidc-with-discovery-secret",
						Namespace: ns,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-provider-conf",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfigRef: &corev1.SecretReference{
						Name:      "oidc-with-provider-conf-secret",
						Namespace: ns,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-neither",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfigRef: &corev1.SecretReference{
						Name:      "oidc-with-neither-secret",
						Namespace: ns,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-both",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfigRef: &corev1.SecretReference{
						Name:      "oidc-with-both-secret",
						Namespace: ns,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "idp-with-no-config",
				},
				Spec: dockyardsv1.IdentityProviderSpec{},
			},
		}

		secrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oidc-with-discovery-secret",
					Namespace: ns,
				},
				Data: toSecretData(t, disc),
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oidc-with-provider-conf-secret",
					Namespace: ns,
				},
				Data: toSecretData(t, conf),
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oidc-with-neither-secret",
					Namespace: ns,
				},
				Data: toSecretData(t, neither),
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oidc-with-both-secret",
					Namespace: ns,
				},
				Data: toSecretData(t, both),
			},
		}

		for i := range secrets {
			secret := secrets[i]

			err := c.Create(ctx, &secret)
			if err != nil {
				t.Fatal(err)
			}

			err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
			if err != nil {
				t.Fatal(err)
			}
		}

		for i := range idps {
			err := c.Create(ctx, &idps[i])
			if err != nil {
				t.Fatal(err)
			}

			err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &idps[i])
			if err != nil {
				t.Fatal(err)
			}
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, reqTarget, nil)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("error reading result body: %s", err)
		}

		var actual []types.IdentityProvider
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling response body: %s", err)
		}

		expected := []types.IdentityProvider{
			{
				ID:   string(idps[0].UID),
				Name: "oidc-with-discovery",
			},
			{
				ID:   string(idps[1].UID),
				Name: "oidc-with-provider-conf",
			},
			{
				ID:   string(idps[3].UID),
				Name: "oidc-with-both",
			},
		}

		sortByID := cmpopts.SortSlices(func(a, b types.IdentityProvider) bool {
			return a.ID < b.ID
		})

		if !cmp.Equal(actual, expected, sortByID) {
			t.Errorf("difference between actual and expected: %s", cmp.Diff(expected, actual, sortByID))
		}
	})
}
