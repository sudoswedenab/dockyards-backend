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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	apitypes "bitbucket.org/sudosweden/dockyards-api/pkg/types"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestOrganizationInvitations_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	userToken := MustSignToken(t, string(user.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	t.Run("test as super user", func(t *testing.T) {
		options := apitypes.InvitationOptions{
			Email: "other@dockyards.dev",
			Role:  string(dockyardsv1.OrganizationMemberRoleUser),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var response apitypes.Invitation
		err = json.Unmarshal(b, &response)
		if err != nil {
			t.Fatal(err)
		}

		objectKey := client.ObjectKey{
			Name:      response.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
		}

		var actual dockyardsv1.Invitation
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{
					Time: response.CreatedAt,
				},
				GenerateName: "pending-",
				Name:         response.Name,
				Namespace:    organization.Spec.NamespaceRef.Name,
				UID:          types.UID(response.ID),
				//
				Finalizers:      actual.Finalizers,
				Generation:      actual.Generation,
				ManagedFields:   actual.ManagedFields,
				ResourceVersion: actual.ResourceVersion,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "other@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		options := apitypes.InvitationOptions{
			Email: "other@dockyards.dev",
			Role:  string(dockyardsv1.OrganizationMemberRoleUser),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		options := apitypes.InvitationOptions{
			Email: "other@dockyards.dev",
			Role:  string(dockyardsv1.OrganizationMemberRoleUser),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test invalid role", func(t *testing.T) {
		options := apitypes.InvitationOptions{
			Email: "other@dockyards.dev",
			Role:  "admin",
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test duration", func(t *testing.T) {
		options := apitypes.InvitationOptions{
			Email:    "other@dockyards.dev",
			Role:     string(dockyardsv1.OrganizationMemberRoleReader),
			Duration: ptr.To("8h"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var response apitypes.Invitation
		err = json.Unmarshal(b, &response)
		if err != nil {
			t.Fatal(err)
		}

		objectKey := client.ObjectKey{
			Name:      response.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
		}

		var actual dockyardsv1.Invitation
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{
					Time: response.CreatedAt,
				},
				GenerateName: "pending-",
				Name:         response.Name,
				Namespace:    organization.Spec.NamespaceRef.Name,
				UID:          types.UID(response.ID),
				//
				Finalizers:      actual.Finalizers,
				Generation:      actual.Generation,
				ManagedFields:   actual.ManagedFields,
				ResourceVersion: actual.ResourceVersion,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "other@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleReader,
				Duration: &metav1.Duration{
					Duration: time.Hour * 8,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestOrganizationInvitations_Delete(t *testing.T) {
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	userToken := MustSignToken(t, string(user.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	t.Run("test as super user", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "delete-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations", invitation.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Invitation
		err = c.Get(ctx, client.ObjectKeyFromObject(&invitation), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if actual.DeletionTimestamp.IsZero() {
			t.Error("expected actual deletion timestamp, got zero")
		}
	})

	t.Run("test as user", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "delete-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations", invitation.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "delete-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations", invitation.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
