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

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)

	superUserToken := MustSignToken(t, string(superUser.UID))

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

		otherUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.OrganizationMemberRoleUser)

		otherUserToken := MustSignToken(t, string(otherUser.UID))

		err := c.Delete(ctx, otherOrganization, client.PropagationPolicy(metav1.DeletePropagationForeground))
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
}
