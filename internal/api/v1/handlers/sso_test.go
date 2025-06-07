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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestGlobalIdentityProviders_List(t *testing.T) {
	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

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
		disc := dockyardsv1.OIDCConfig{OIDCProviderDiscoveryURL: new(string)}
		conf := dockyardsv1.OIDCConfig{
			OIDCProviderConfig: &dockyardsv1.OIDCProviderConfig{
				IDTokenSigningAlgs: []string{
					"RS256",
				},
			},
		}
		both := dockyardsv1.OIDCConfig{
			OIDCProviderConfig: &dockyardsv1.OIDCProviderConfig{
				IDTokenSigningAlgs: []string{
					"RS256",
				},
			},
			OIDCProviderDiscoveryURL: new(string),
		}

		idps := []dockyardsv1.IdentityProvider{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-discovery",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfig: &disc,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-provider-conf",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfig: &conf,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-neither",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfig: new(dockyardsv1.OIDCConfig),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oidc-with-both",
				},
				Spec: dockyardsv1.IdentityProviderSpec{
					OIDCConfig: &both,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "idp-with-no-config",
				},
				Spec: dockyardsv1.IdentityProviderSpec{},
			},
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
