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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterWorkloads_Create(t *testing.T) {
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

	organization := testEnvironment.GetOrganization()
	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	dockyardsNamespace := testEnvironment.GetDockyardsNamespace()

	h := handler{
		Client:    mgr.GetClient(),
		namespace: dockyardsNamespace,
	}

	handlerFunc := CreateClusterResource(&h, "workloads", h.CreateClusterWorkload)

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
		},
	}

	err = c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

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
		request := types.Workload{
			Name:                 ptr.To("test-user"),
			WorkloadTemplateName: ptr.To("test"),
			Namespace:            ptr.To("testing"),
		}

		b, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

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
		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test no workload template name", func(t *testing.T) {
		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

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

		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

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

		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

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

		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc.ServeHTTP(w, r.Clone(ctx))

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

	organization := testEnvironment.GetOrganization()
	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	h := handler{
		Client:    mgr.GetClient(),
		namespace: testEnvironment.GetDockyardsNamespace(),
	}

	handlerFunc := DeleteClusterResource(&h, "workloads", h.DeleteClusterWorkload)

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
		},
	}

	err = c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test as super user", func(t *testing.T) {
		workloadName := "test-super-user"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Status.NamespaceRef.Name,
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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
				Namespace: organization.Status.NamespaceRef.Name,
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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
				Namespace: organization.Status.NamespaceRef.Name,
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", "test-non-existing")

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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

	organization := testEnvironment.GetOrganization()
	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	h := handler{
		Client:    mgr.GetClient(),
		namespace: testEnvironment.GetDockyardsNamespace(),
	}

	handlerFunc := UpdateClusterResource(&h, "workloads", h.UpdateClusterWorkload)

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
		},
	}

	err = c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test as super user", func(t *testing.T) {
		workloadName := "test-super-user"

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + workloadName,
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
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
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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
		r := httptest.NewRequest(http.MethodDelete, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
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
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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
		r := httptest.NewRequest(http.MethodDelete, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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
		r := httptest.NewRequest(http.MethodDelete, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
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
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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
		r := httptest.NewRequest(http.MethodDelete, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
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
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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
		r := httptest.NewRequest(http.MethodDelete, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

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
				Namespace: organization.Status.NamespaceRef.Name,
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

		request := types.Workload{
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

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("resourceName", workloadName)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, statusCode)
		}
	})
}
