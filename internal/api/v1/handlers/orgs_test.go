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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	apitypes "bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.TODO())

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		t.Fatal(err)
	}

	organization := testEnvironment.GetOrganization()
	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	h := handler{
		Client: mgr.GetClient(),
	}

	handlerFunc := ListGlobalResource("organizations", h.ListGlobalOrganizations)

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	mgr.GetCache().WaitForCacheSync(ctx)

	less := func(a, b apitypes.Organization) bool {
		return a.CreatedAt.Before(b.CreatedAt)
	}

	sortSlices := cmpopts.SortSlices(less)

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

		u := url.URL{
			Path: path.Join("/v1/orgs"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		ctx := middleware.ContextWithSubject(context.Background(), string(otherUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

func TestGlobalOrganization_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.TODO())

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		t.Fatal(err)
	}

	reader := testEnvironment.GetReader()

	h := handler{
		Client: mgr.GetClient(),
	}

	handlerFunc := CreateGlobalResource("organizations", h.CreateGlobalOrganization)

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

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

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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
							Name: reader.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  reader.UID,
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

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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
							Name: reader.Name,
						},
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  reader.UID,
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
