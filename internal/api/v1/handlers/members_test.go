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
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"k8s.io/utils/ptr"
)

func TestOrganizationMembers_List(t *testing.T) {
	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	userToken := MustSignToken(t, string(user.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	expected := []types.Member{
		{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(superUser.UID),
			Name:      superUser.Name,
			Role:      ptr.To(string(dockyardsv1.OrganizationMemberRoleSuperUser)),
		},
		{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(user.UID),
			Name:      user.Name,
			Role:      ptr.To(string(dockyardsv1.OrganizationMemberRoleUser)),
		},
		{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(reader.UID),
			Name:      reader.Name,
			Role:      ptr.To(string(dockyardsv1.OrganizationMemberRoleReader)),
		},
	}

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "members"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Member
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "members"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Member
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "members"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Member
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test other user", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)
		otherUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.OrganizationMemberRoleUser)
		otherUserToken := MustSignToken(t, string(otherUser.UID))

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "members"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
