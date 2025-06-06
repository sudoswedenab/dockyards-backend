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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterWorkloads_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	dockyardsNamespace := testEnvironment.GetDockyardsNamespace()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		request := types.WorkloadOptions{
			Name:                 ptr.To("test-super-user"),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, statusCode)

			return
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-test-super-user",
			Namespace: cluster.Namespace,
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Error(err)

			return
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-super-user",
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
				UID:               actual.UID,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "testing",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "test",
					Namespace: &dockyardsNamespace,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		request := types.WorkloadOptions{
			Name:                 ptr.To("test-user"),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, statusCode)

			return
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-test-user",
			Namespace: cluster.Namespace,
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Error(err)

			return
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-user",
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
				UID:               actual.UID,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "testing",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "test",
					Namespace: &dockyardsNamespace,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		request := types.WorkloadOptions{
			Name:                 ptr.To("test-user"),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
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

	t.Run("test no workload template name", func(t *testing.T) {
		request := types.WorkloadOptions{
			Name:      ptr.To("test-super-user"),
			Namespace: ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test input", func(t *testing.T) {
		input := map[string]any{
			"qwfp": "arst",
			"neio": 5,
			"zxcv": map[string]any{
				"test": true,
			},
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To("test-input"),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
			Input:                &input,
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)

			return
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-test-input",
			Namespace: cluster.Namespace,
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		raw, err := json.Marshal(&input)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-input",
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
				UID:               actual.UID,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "testing",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "test",
					Namespace: &dockyardsNamespace,
				},
				Input: &apiextensionsv1.JSON{
					Raw: raw,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test namespace", func(t *testing.T) {
		name := "test-namespace"

		request := types.WorkloadOptions{
			Name:                 &name,
			WorkloadTemplateName: ptr.To("test"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, statusCode)

			return
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-" + name,
			Namespace: cluster.Namespace,
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + name,
				Namespace: cluster.Namespace,
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
				UID:               actual.UID,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: name,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "test",
					Namespace: &dockyardsNamespace,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test already exists", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-already-exists",
				Namespace: cluster.Namespace,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience: dockyardsv1.ProvenienceUser,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To("test-already-exists"),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusConflict {
			t.Errorf("expected status code %d, got %d", http.StatusConflict, statusCode)

			return
		}

	})
}

func TestClusterWorkloads_Delete(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test as super user", func(t *testing.T) {
		workloadName := "test-super-user"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		workloadName := "test-user"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		workloadName := "test-reader"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
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

	t.Run("test non-existing workload", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", "test-non-existing"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status code %d, got %d", http.StatusNotFound, statusCode)
		}
	})
}

func TestClusterWorkloads_Update(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test as super user", func(t *testing.T) {
		workloadName := "test-super-user"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("update"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, client.ObjectKeyFromObject(&workload), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:              workload.Name,
				Namespace:         workload.Namespace,
				UID:               workload.UID,
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "update",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		workloadName := "test-user"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("update"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, client.ObjectKeyFromObject(&workload), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:              workload.Name,
				Namespace:         workload.Namespace,
				UID:               workload.UID,
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "update",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		workloadName := "test-reader"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("update"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test input", func(t *testing.T) {
		workloadName := "test-input"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
				Input: &apiextensionsv1.JSON{
					Raw: []byte(`{"replicas":1}`),
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
			Input: &map[string]any{
				"replicas": 2,
			},
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, client.ObjectKeyFromObject(&workload), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:              workload.Name,
				Namespace:         workload.Namespace,
				UID:               workload.UID,
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "testing",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
				Input: &apiextensionsv1.JSON{
					Raw: []byte(`{"replicas":2}`),
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test remove input", func(t *testing.T) {
		workloadName := "test-remove-input"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
				Input: &apiextensionsv1.JSON{
					Raw: []byte(`{"replicas":1}`),
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, client.ObjectKeyFromObject(&workload), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:              workload.Name,
				Namespace:         workload.Namespace,
				UID:               workload.UID,
				CreationTimestamp: actual.CreationTimestamp,
				Generation:        actual.Generation,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceUser,
				TargetNamespace: "testing",
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test workload template name", func(t *testing.T) {
		workloadName := "test-workload-template-name"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("update"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test dockyards provenience", func(t *testing.T) {
		workloadName := "test-dockyards-provenience"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				TargetNamespace: "testing",
				Provenience:     dockyardsv1.ProvenienceDockyards,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind: dockyardsv1.WorkloadTemplateKind,
					Name: "test",
				},
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workload)
		if err != nil {
			t.Fatal(err)
		}

		request := types.WorkloadOptions{
			Name:                 ptr.To(workloadName),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("update"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, statusCode)
		}
	})
}

func TestClusterWorkloads_Get(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	readerToken := MustSignToken(t, reader.Name)

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test urls", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: cluster.Name + "-test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience:     dockyardsv1.ProvenienceDockyards,
				TargetNamespace: "testing",
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		patch := client.MergeFrom(workload.DeepCopy())

		workload.Status.URLs = []string{
			"http://testing.dockyards.dev",
		}

		err = c.Status().Patch(ctx, &workload, patch)
		if err != nil {
			t.Fatal(err)
		}

		workloadName := strings.TrimPrefix(workload.Name, cluster.Name+"-")

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := mgr.GetClient().Get(ctx, client.ObjectKeyFromObject(&workload), &workload)
			if err != nil {
				return true, err
			}

			return len(workload.Status.URLs) > 0, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", workloadName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.Workload
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.Workload{
			ID:        string(workload.UID),
			Namespace: &workload.Spec.TargetNamespace,
			Name:      workloadName,
			URLs:      &workload.Status.URLs,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
