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
	"net/url"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
)

func TestGlobalTokens_Get(t *testing.T) {
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)
	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	superUserToken := MustSignRefreshToken(t, superUser.Name)

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: "/v1/refresh",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.Tokens
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("access: %s", actual.AccessToken)

		token, err := jwt.ParseWithClaims(actual.AccessToken, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
			return &accessKey.PublicKey, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		subject, err := token.Claims.(*jwt.RegisteredClaims).GetSubject()
		if err != nil {
			t.Fatal(err)
		}

		if subject != superUser.Name {
			t.Errorf("expected access token subject %s, got %s", superUser.Name, subject)
		}

		token, err = jwt.ParseWithClaims(actual.RefreshToken, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
			return &refreshKey.PublicKey, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		subject, err = token.Claims.(*jwt.RegisteredClaims).GetSubject()
		if err != nil {
			t.Fatal(err)
		}

		if subject != superUser.Name {
			t.Errorf("expected refresh token subject %s, got %s", superUser.Name, subject)
		}
	})

	t.Run("test access token", func(t *testing.T) {
		superUserToken := MustSignToken(t, superUser.Name)

		u := url.URL{
			Path: "/v1/refresh",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test deleted user", func(t *testing.T) {
		user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
		userToken := MustSignRefreshToken(t, user.Name)

		err := c.Delete(ctx, user)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: "/v1/refresh",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
