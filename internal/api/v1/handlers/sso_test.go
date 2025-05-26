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

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestListIdentityProviders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reqTarget := "/v1/identity-providers"

	t.Run("test no providers", func(t *testing.T) {
		scheme := scheme.Scheme
		dockyardsv1.AddToScheme(scheme)

		idps := dockyardsv1.IdentityProviderList{}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&idps).Build()
		h := handler{Client: fakeClient}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, reqTarget, nil)
		ctx := middleware.ContextWithLogger(context.Background(), logger)

		h.ListIdentityProviders(w, r.Clone(ctx))

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
		scheme := scheme.Scheme
		dockyardsv1.AddToScheme(scheme)

		disc := dockyardsv1.OIDCConfig{OIDCProviderDiscoveryURL: new(string)}
		conf := dockyardsv1.OIDCConfig{OIDCProviderConfig: new(dockyardsv1.OIDCProviderConfig)}
		both := dockyardsv1.OIDCConfig{
			OIDCProviderConfig:       new(dockyardsv1.OIDCProviderConfig),
			OIDCProviderDiscoveryURL: new(string),
		}

		idps := dockyardsv1.IdentityProviderList{
			Items: []dockyardsv1.IdentityProvider{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "oidc-with-discovery",
						UID:  "1",
					},
					Spec: dockyardsv1.IdentityProviderSpec{
						OIDCConfig: &disc,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "oidc-with-provider-conf",
						UID:  "2",
					},
					Spec: dockyardsv1.IdentityProviderSpec{
						OIDCConfig: &conf,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "oidc-with-neither",
						UID:  "3",
					},
					Spec: dockyardsv1.IdentityProviderSpec{
						OIDCConfig: new(dockyardsv1.OIDCConfig),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "oidc-with-both",
						UID:  "4",
					},
					Spec: dockyardsv1.IdentityProviderSpec{
						OIDCConfig: &both,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "idp-with-no-config",
						UID:  "5",
					},
					Spec: dockyardsv1.IdentityProviderSpec{},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&idps).Build()
		h := handler{Client: fakeClient}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, reqTarget, nil)
		ctx := middleware.ContextWithLogger(context.Background(), logger)

		h.ListIdentityProviders(w, r.Clone(ctx))

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
				ID:   "1",
				Name: "oidc-with-discovery",
			},
			{
				ID:   "2",
				Name: "oidc-with-provider-conf",
			},
			{
				ID:   "4",
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
