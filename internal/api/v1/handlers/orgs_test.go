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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	apitypes "github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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
	c := mgr.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

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
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "other-",
			},
		}

		err := c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		otherOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "other-",
			},
			Spec: dockyardsv1.OrganizationSpec{
				DisplayName: "testing",
				NamespaceRef: &corev1.LocalObjectReference{
					Name: namespace.Name,
				},
			},
		}

		err = c.Create(ctx, &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		member := dockyardsv1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					dockyardsv1.LabelOrganizationName: otherOrganization.Name,
					dockyardsv1.LabelUserName:         reader.Name,
				},
				Name:      reader.Name,
				Namespace: namespace.Name,
			},
			Spec: dockyardsv1.MemberSpec{
				Role: dockyardsv1.RoleSuperUser,
			},
		}

		err = c.Create(ctx, &member)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &member)
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
				ID:          string(otherOrganization.UID),
				Name:        otherOrganization.Name,
				DisplayName: &otherOrganization.Spec.DisplayName,
				CreatedAt:   otherOrganization.CreationTimestamp.Time,
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

		otherUserToken := MustSignToken(t, otherUser.Name)

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

	t.Run("test deleted organization", func(t *testing.T) {
		deletedNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "deleted-",
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
			},
		}

		err := c.Create(ctx, &deletedNamespace)
		if err != nil {
			t.Fatal(err)
		}

		deletedOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "deleted-",
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
			},
			Spec: dockyardsv1.OrganizationSpec{
				NamespaceRef: &corev1.LocalObjectReference{
					Name: deletedNamespace.Name,
				},
			},
		}

		err = c.Create(ctx, &deletedOrganization)
		if err != nil {
			t.Fatal(err)
		}

		member := dockyardsv1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
				Labels: map[string]string{
					dockyardsv1.LabelOrganizationName: deletedOrganization.Name,
					dockyardsv1.LabelUserName:         superUser.Name,
				},
				Name:      superUser.Name,
				Namespace: deletedNamespace.Name,
			},
			Spec: dockyardsv1.MemberSpec{
				UserRef: corev1.TypedLocalObjectReference{
					Kind: dockyardsv1.UserKind,
					Name: superUser.Name,
				},
				Role: dockyardsv1.RoleSuperUser,
			},
		}

		err = c.Create(ctx, &member)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		err = c.Delete(ctx, &deletedOrganization)
		if err != nil {
			t.Fatal(err)
		}

		for range 10 {
			err := c.Get(ctx, client.ObjectKeyFromObject(&deletedOrganization), &deletedOrganization)
			if err != nil {
				t.Fatal(err)
			}

			if !deletedOrganization.DeletionTimestamp.IsZero() {
				break
			}

			time.Sleep(time.Second)
		}

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
			{
				ID:        string(deletedOrganization.UID),
				Name:      deletedOrganization.Name,
				CreatedAt: deletedOrganization.CreationTimestamp.Time,
				DeletedAt: &deletedOrganization.DeletionTimestamp.Time,
			},
		}

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

	otherUserToken := MustSignToken(t, otherUser.Name)

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherUser)
	if err != nil {
		t.Fatal(err)
	}

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
				Labels: map[string]string{
					dockyardsv1.LabelOrganizationName: organization.Name,
				},
				UID: types.UID(organization.ID),
			},
			Spec: dockyardsv1.OrganizationSpec{
				NamespaceRef: &corev1.LocalObjectReference{
					Name: organization.Name,
				},
				ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
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

		expectedMember := dockyardsv1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					dockyardsv1.LabelRoleName:         dockyardsv1.RoleSuperUser,
					dockyardsv1.LabelOrganizationName: organization.Name,
					dockyardsv1.LabelUserName:         otherUser.Name,
				},
				Name:      otherUser.Name,
				Namespace: actual.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.MemberSpec{
				Role: dockyardsv1.RoleSuperUser,
				UserRef: corev1.TypedLocalObjectReference{
					APIGroup: &dockyardsv1.GroupVersion.Group,
					Kind:     dockyardsv1.UserKind,
					Name:     otherUser.Name,
				},
			},
		}

		var actualMember dockyardsv1.Member
		err = c.Get(ctx, client.ObjectKeyFromObject(&expectedMember), &actualMember)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actualMember, expectedMember, ignoreFields, ignoreObjectMetaFields) {
			t.Errorf("diff: %s", cmp.Diff(expectedMember, actualMember, ignoreFields, ignoreObjectMetaFields))
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
				Labels: map[string]string{
					dockyardsv1.LabelOrganizationName: organization.Name,
				},
			},
			Spec: dockyardsv1.OrganizationSpec{
				DisplayName: "testing",
				NamespaceRef: &corev1.LocalObjectReference{
					Name: organization.Name,
				},
				ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
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

	t.Run("test voucher code", func(t *testing.T) {
		organizationVoucher := dockyardsv1.OrganizationVoucher{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    testEnvironment.GetDockyardsNamespace(),
			},
			Spec: dockyardsv1.OrganizationVoucherSpec{
				Code: "TEST-123",
				PoolRef: &corev1.TypedObjectReference{
					Kind: "TestPool",
					Name: "testing",
				},
			},
		}

		err := c.Create(ctx, &organizationVoucher)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &organizationVoucher)
		if err != nil {
			t.Fatal(err)
		}

		req := apitypes.Organization{
			VoucherCode: &organizationVoucher.Spec.Code,
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
				Annotations: map[string]string{
					dockyardsv1.AnnotationVoucherCode: organizationVoucher.Spec.Code,
				},
				Name: organization.Name,
				CreationTimestamp: metav1.Time{
					Time: organization.CreatedAt,
				},
				UID: types.UID(organization.ID),
				Labels: map[string]string{
					dockyardsv1.LabelOrganizationName: organization.Name,
				},
			},
			Spec: dockyardsv1.OrganizationSpec{
				NamespaceRef: &corev1.LocalObjectReference{
					Name: organization.Name,
				},
				ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
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

	t.Run("test redeemed voucher code", func(t *testing.T) {
		organizationVoucher := dockyardsv1.OrganizationVoucher{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    testEnvironment.GetDockyardsNamespace(),
			},
			Spec: dockyardsv1.OrganizationVoucherSpec{
				Code: "TEST-REDEEMED",
				PoolRef: &corev1.TypedObjectReference{
					Kind: "TestPool",
					Name: "testing",
				},
			},
		}

		err := c.Create(ctx, &organizationVoucher)
		if err != nil {
			t.Fatal(err)
		}

		patch := client.MergeFrom(organizationVoucher.DeepCopy())

		organizationVoucher.Status.Redeemed = true

		err = c.Status().Patch(ctx, &organizationVoucher, patch)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &organizationVoucher)
		if err != nil {
			t.Fatal(err)
		}

		req := apitypes.Organization{
			VoucherCode: &organizationVoucher.Spec.Code,
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
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid voucher code", func(t *testing.T) {
		req := apitypes.Organization{
			VoucherCode: ptr.To("TEST-INVALID"),
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
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
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

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

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

func TestGlobalOrganizations_Get(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	patch := client.MergeFrom(organization.DeepCopy())

	organization.Spec.DisplayName = "testing"

	err := c.Patch(ctx, organization, patch)
	if err != nil {
		t.Fatal(err)
	}

	readyCondition := metav1.Condition{
		Type:               dockyardsv1.ReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             dockyardsv1.ReadyReason,
		Message:            "testing",
		LastTransitionTime: metav1.Now(),
	}

	meta.SetStatusCondition(&organization.Status.Conditions, readyCondition)

	err = c.Status().Patch(ctx, organization, patch)
	if err != nil {
		t.Fatal(err)
	}

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)

	superUserToken := MustSignToken(t, superUser.Name)

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), organization)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
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

		var actual apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := apitypes.Organization{
			Condition:   &readyCondition.Reason,
			CreatedAt:   organization.CreationTimestamp.Time,
			DisplayName: &organization.Spec.DisplayName,
			ID:          string(organization.UID),
			Name:        organization.Name,
			ProviderID:  organization.Spec.ProviderID,
			UpdatedAt:   ptr.To(readyCondition.LastTransitionTime.Time.Truncate(time.Second)),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test other organization", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test deleted organization", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)

		patch := client.MergeFrom(otherOrganization.DeepCopy())

		otherOrganization.Finalizers = []string{
			"backend.dockyards.io/testing",
		}

		err := c.Patch(ctx, otherOrganization, patch)
		if err != nil {
			t.Fatal(err)
		}

		otherUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.RoleUser)

		otherUserToken := MustSignToken(t, otherUser.Name)

		err = c.Delete(ctx, otherOrganization, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil {
			t.Fatal(err)
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(otherOrganization), otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name),
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

		var actual apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := apitypes.Organization{
			ID:         string(otherOrganization.UID),
			Name:       otherOrganization.Name,
			ProviderID: otherOrganization.Spec.ProviderID,
			CreatedAt:  otherOrganization.CreationTimestamp.Time,
			DeletedAt:  &otherOrganization.DeletionTimestamp.Time,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test credential reference", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)

		otherUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.RoleUser)

		otherUserToken := MustSignToken(t, otherUser.Name)

		patch := client.MergeFrom(otherOrganization.DeepCopy())

		otherOrganization.Spec.CredentialRef = &corev1.TypedObjectReference{
			Kind: "Secret",
			Name: "credential-testing",
		}

		err := c.Patch(ctx, otherOrganization, patch)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name),
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

		var actual apitypes.Organization
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := apitypes.Organization{
			ID:                      string(otherOrganization.UID),
			Name:                    otherOrganization.Name,
			ProviderID:              otherOrganization.Spec.ProviderID,
			CreatedAt:               otherOrganization.CreationTimestamp.Time,
			CredentialReferenceName: ptr.To("testing"),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestGlobalOrganizations_Update(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	err := testingutil.RetryUntilFound(ctx, mgr.GetClient(), organization)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		options := apitypes.OrganizationOptions{
			DisplayName: ptr.To("testing"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Organization
		err = c.Get(ctx, client.ObjectKeyFromObject(organization), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Organization{
			ObjectMeta: actual.ObjectMeta,
			Spec: dockyardsv1.OrganizationSpec{
				DisplayName: "testing",
				//
				NamespaceRef: organization.Spec.NamespaceRef,
				ProviderID:   organization.Spec.ProviderID,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		options := apitypes.OrganizationOptions{
			DisplayName: ptr.To("testing"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		options := apitypes.OrganizationOptions{
			DisplayName: ptr.To("testing"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test voucher code", func(t *testing.T) {
		options := apitypes.OrganizationOptions{
			VoucherCode: ptr.To("TST-123"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test credential reference", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)
		otherSuperUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.RoleSuperUser)
		otherSuperUserToken := MustSignToken(t, otherSuperUser.Name)

		options := apitypes.OrganizationOptions{
			CredentialReferenceName: ptr.To("testing"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+otherSuperUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Organization
		err = c.Get(ctx, client.ObjectKeyFromObject(otherOrganization), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Organization{
			ObjectMeta: actual.ObjectMeta,
			Spec: dockyardsv1.OrganizationSpec{
				CredentialRef: &corev1.TypedObjectReference{
					Kind:      "Secret",
					Name:      "credential-testing",
					Namespace: &otherOrganization.Spec.NamespaceRef.Name,
				},
				//
				NamespaceRef: otherOrganization.Spec.NamespaceRef,
				ProviderID:   otherOrganization.Spec.ProviderID,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test duration", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)
		otherSuperUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.RoleSuperUser)
		otherSuperUserToken := MustSignToken(t, otherSuperUser.Name)

		options := apitypes.OrganizationOptions{
			Duration: ptr.To("15m"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+otherSuperUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Organization
		err = c.Get(ctx, client.ObjectKeyFromObject(otherOrganization), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Organization{
			ObjectMeta: actual.ObjectMeta,
			Spec: dockyardsv1.OrganizationSpec{
				Duration: &metav1.Duration{
					Duration: time.Minute * 15,
				},
				//
				NamespaceRef: otherOrganization.Spec.NamespaceRef,
				ProviderID:   otherOrganization.Spec.ProviderID,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
