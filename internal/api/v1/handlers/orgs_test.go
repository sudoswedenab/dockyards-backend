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

func TestGlobalOrganizations_List(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	userToken := MustSignToken(t, string(user.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	mgr.GetCache().WaitForCacheSync(ctx)

	less := func(a, b apitypes.Organization) bool {
		return a.Name < b.Name
	}

	sortSlices := cmpopts.SortSlices(less)

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs"),
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

		var actual []apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []apitypes.Organization{
			{
				ID:        string(organization.UID),
				Name:      organization.Name,
				CreatedAt: organization.CreationTimestamp.Time,
			},
		}

		if !cmp.Equal(actual, expected, sortSlices) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, sortSlices))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs"),
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

		var actual []apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []apitypes.Organization{
			{
				ID:        string(organization.UID),
				Name:      organization.Name,
				CreatedAt: organization.CreationTimestamp.Time,
			},
		}

		if !cmp.Equal(actual, expected, sortSlices) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, sortSlices))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs"),
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

		var actual []apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []apitypes.Organization{
			{
				ID:        string(organization.UID),
				Name:      organization.Name,
				CreatedAt: organization.CreationTimestamp.Time,
			},
		}

		if !cmp.Equal(actual, expected, sortSlices) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, sortSlices))
		}
	})

	t.Run("test multiple membership", func(t *testing.T) {
		otherOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "other-",
			},
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							APIGroup: &dockyardsv1.GroupVersion.Group,
							Kind:     dockyardsv1.UserKind,
							Name:     user.Name,
						},
						UID:  reader.UID,
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
					},
				},
			},
		}

		err := c.Create(ctx, &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs"),
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

		var actual []apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []apitypes.Organization{
			{
				ID:        string(organization.UID),
				Name:      organization.Name,
				CreatedAt: organization.CreationTimestamp.Time,
			},
			{
				ID:        string(otherOrganization.UID),
				Name:      otherOrganization.Name,
				CreatedAt: otherOrganization.CreationTimestamp.Time,
			},
		}

		if !cmp.Equal(actual, expected, sortSlices) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, sortSlices))
		}
	})

	t.Run("test no membership", func(t *testing.T) {
		otherUser := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "other-",
			},
		}

		err := c.Create(ctx, &otherUser)
		if err != nil {
			t.Fatal(err)
		}

		otherUserToken, err := SignToken(string(otherUser.UID))
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs"),
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

		var actual []apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []apitypes.Organization{}

		if !cmp.Equal(actual, expected, sortSlices) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, sortSlices))
		}
	})
}

func TestGlobalOrganizations_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	otherUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "other-",
		},
	}

	err := c.Create(ctx, &otherUser)
	if err != nil {
		t.Fatal(err)
	}

	otherUserToken, err := SignToken(string(otherUser.UID))
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	ignoreFields := cmpopts.IgnoreFields(metav1.ObjectMeta{}, "Generation", "ResourceVersion", "ManagedFields")

	t.Run("test empty request", func(t *testing.T) {
		req := apitypes.Organization{}

		b, err := json.Marshal(&req)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var organization apitypes.Organization
		err = json.Unmarshal(b, &organization)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: organization.Name,
				CreationTimestamp: metav1.Time{
					Time: organization.CreatedAt,
				},
				UID: types.UID(organization.ID),
			},
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							Kind: dockyardsv1.UserKind,
							Name: otherUser.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  otherUser.UID,
					},
				},
				NamespaceRef: &corev1.LocalObjectReference{
					Name: organization.Name,
				},
			},
		}

		var actual dockyardsv1.Organization
		err = c.Get(ctx, client.ObjectKeyFromObject(&expected), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected, ignoreFields) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, ignoreFields))
		}

		expectedNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
				Name:         actual.Spec.NamespaceRef.Name,
				Labels: map[string]string{
					dockyardsv1.LabelOrganizationName: organization.Name,
					corev1.LabelMetadataName:          organization.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       actual.Name,
						UID:        actual.UID,
					},
				},
			},
		}

		var actualNamespace corev1.Namespace
		err = c.Get(ctx, client.ObjectKeyFromObject(&expectedNamespace), &actualNamespace)
		if err != nil {
			t.Fatal(err)
		}

		ignoreObjectMetaFields := cmpopts.IgnoreFields(metav1.ObjectMeta{}, "CreationTimestamp", "UID")
		ignoreNamespaceFields := cmpopts.IgnoreFields(corev1.Namespace{}, "Spec", "Status")

		if !cmp.Equal(actualNamespace, expectedNamespace, ignoreFields, ignoreObjectMetaFields, ignoreNamespaceFields) {
			t.Errorf("diff: %s", cmp.Diff(expectedNamespace, actualNamespace, ignoreFields, ignoreObjectMetaFields, ignoreNamespaceFields))
		}
	})

	t.Run("test display name", func(t *testing.T) {
		req := apitypes.Organization{
			DisplayName: ptr.To("testing"),
		}

		b, err := json.Marshal(&req)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var organization apitypes.Organization
		err = json.Unmarshal(b, &organization)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				Name: organization.Name,
				CreationTimestamp: metav1.Time{
					Time: organization.CreatedAt,
				},
				UID: types.UID(organization.ID),
			},
			Spec: dockyardsv1.OrganizationSpec{
				DisplayName: "testing",
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						TypedLocalObjectReference: corev1.TypedLocalObjectReference{
							Kind: dockyardsv1.UserKind,
							Name: otherUser.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  otherUser.UID,
					},
				},
				NamespaceRef: &corev1.LocalObjectReference{
					Name: organization.Name,
				},
			},
		}

		var actual dockyardsv1.Organization
		err = c.Get(ctx, client.ObjectKeyFromObject(&expected), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected, ignoreFields) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, ignoreFields))
		}
	})
}

func TestGlobalOrganizations_Delete(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	userToken := MustSignToken(t, string(user.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}

		var actual dockyardsv1.Organization
		err := c.Get(ctx, client.ObjectKeyFromObject(organization), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !actual.DeletionTimestamp.IsZero() {
			t.Error("expected zero deletion timestamp")
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}

		var actual dockyardsv1.Organization
		err := c.Get(ctx, client.ObjectKeyFromObject(organization), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !actual.DeletionTimestamp.IsZero() {
			t.Error("expected zero deletion timestamp")
		}
	})

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Organization
		err := c.Get(ctx, client.ObjectKeyFromObject(organization), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if actual.DeletionTimestamp.IsZero() {
			t.Error("expected deletion timestamp, got zero")
		}
	})
}
