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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
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

func TestOrganizationInvitations_List(t *testing.T) {
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

	invitations := []dockyardsv1.Invitation{
		{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "pending-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "test@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "pending-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "duration@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleSuperUser,
				Duration: &metav1.Duration{
					Duration: time.Minute * 15,
				},
			},
		},
	}

	expected := make([]apitypes.Invitation, len(invitations))

	for i, invitation := range invitations {
		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		expected[i] = apitypes.Invitation{
			CreatedAt: invitation.CreationTimestamp.Time,
			ID:        string(invitation.UID),
			Name:      invitation.Name,
			Role:      string(invitation.Spec.Role),
		}

		if invitation.Spec.Duration != nil {
			expected[i].Duration = ptr.To(invitation.Spec.Duration.String())
			expected[i].ExpiresAt = &invitation.GetExpiration().Time
		}
	}

	byID := cmpopts.SortSlices(func(a, b apitypes.Invitation) bool {
		return a.ID < b.ID
	})

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
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

		var actual []apitypes.Invitation
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected, byID) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, byID))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
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

		var actual []apitypes.Invitation
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected, byID) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, byID))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "invitations"),
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

		var actual []apitypes.Invitation
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected, byID) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, byID))
		}
	})
}

func TestGlobalInvitations_List(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	mgr.GetCache().WaitForCacheSync(ctx)

	otherUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "test@dockyards.dev",
		},
	}

	err := c.Create(ctx, &otherUser)
	if err != nil {
		t.Fatal(err)
	}

	otherUserToken := MustSignToken(t, string(otherUser.UID))

	t.Run("test as other user", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: otherUser.Spec.Email,
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		otherInvitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "other@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		}

		err = c.Create(ctx, &otherInvitation)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherInvitation)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/invitations"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []apitypes.Invitation
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []apitypes.Invitation{
			{
				CreatedAt:        invitation.CreationTimestamp.Time,
				ID:               string(invitation.UID),
				Name:             invitation.Name,
				OrganizationName: &organization.Name,
				Role:             string(dockyardsv1.OrganizationMemberRoleUser),
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Fatalf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestGlobalInvitations_Delete(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	mgr.GetCache().WaitForCacheSync(ctx)

	otherUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "test@dockyards.dev",
		},
	}

	err := c.Create(ctx, &otherUser)
	if err != nil {
		t.Fatal(err)
	}

	otherUserToken := MustSignToken(t, string(otherUser.UID))

	t.Run("test as other user", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: otherUser.Spec.Email,
				Role:  dockyardsv1.OrganizationMemberRoleReader,
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &invitation)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/invitations", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})
}

func TestGlobalInvitations_Update(t *testing.T) {
	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	otherUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "other@dockyards.dev",
		},
	}

	err := c.Create(ctx, &otherUser)
	if err != nil {
		t.Fatal(err)
	}

	otherUserToken := MustSignToken(t, string(otherUser.UID))

	byUID := cmpopts.SortSlices(func(a, b dockyardsv1.OrganizationMemberReference) bool {
		return a.UID < b.UID
	})

	t.Run("test without invitation", func(t *testing.T) {
		options := apitypes.InvitationOptions{}

		b, err := json.Marshal(&options)

		if err != nil {
			t.Fatal(err)
		}
		u := url.URL{
			Path: path.Join("/v1/invitations", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test as other user", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
				GenerateName: "pending-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: otherUser.Spec.Email,
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &invitation)
		if err != nil {
			t.Fatal(err)
		}

		options := apitypes.InvitationOptions{}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/invitations", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actualInvitation dockyardsv1.Invitation
		err = c.Get(ctx, client.ObjectKeyFromObject(&invitation), &actualInvitation)
		if err != nil {
			t.Fatal(err)
		}

		if actualInvitation.DeletionTimestamp.IsZero() {
			t.Error("expected actual invitation deletion timestamp, got zero")
		}

		var actualOrganization dockyardsv1.Organization
		err = c.Get(ctx, client.ObjectKeyFromObject(organization), &actualOrganization)
		if err != nil {
			t.Fatal(err)
		}

		expectedOrganization := dockyardsv1.Organization{
			ObjectMeta: actualOrganization.ObjectMeta,
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							APIGroup: &dockyardsv1.GroupVersion.Group,
							Kind:     dockyardsv1.UserKind,
							Name:     otherUser.Name,
						},
						Role: invitation.Spec.Role,
						UID:  otherUser.UID,
					},
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							APIGroup: &dockyardsv1.GroupVersion.Group,
							Kind:     dockyardsv1.UserKind,
							Name:     superUser.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  superUser.UID,
					},
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							APIGroup: &dockyardsv1.GroupVersion.Group,
							Kind:     dockyardsv1.UserKind,
							Name:     user.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleUser,
						UID:  user.UID,
					},
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							APIGroup: &dockyardsv1.GroupVersion.Group,
							Kind:     dockyardsv1.UserKind,
							Name:     reader.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleReader,
						UID:  reader.UID,
					},
				},
				NamespaceRef: actualOrganization.Spec.NamespaceRef,
				ProviderID:   actualOrganization.Spec.ProviderID,
			},
		}

		if !cmp.Equal(actualOrganization, expectedOrganization, byUID) {
			t.Errorf("diff: %s", cmp.Diff(expectedOrganization, actualOrganization, byUID))
		}
	})
}
