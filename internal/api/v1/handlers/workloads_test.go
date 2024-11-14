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
	"time"

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
			WorkloadTemplateName: ptr.To("testing"),
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

		h.CreateClusterWorkload(w, r.Clone(ctx))

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
				Provenience: dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "testing",
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
			WorkloadTemplateName: ptr.To("testing"),
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

		h.CreateClusterWorkload(w, r.Clone(ctx))

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
				Provenience: dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "testing",
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
			WorkloadTemplateName: ptr.To("testing"),
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

		h.CreateClusterWorkload(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test no workload template name", func(t *testing.T) {
		request := types.Workload{
			Name: ptr.To("test-super-user"),
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

		h.CreateClusterWorkload(w, r.Clone(ctx))

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
			WorkloadTemplateName: ptr.To("testing"),
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

		h.CreateClusterWorkload(w, r.Clone(ctx))

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
				Provenience: dockyardsv1.ProvenienceUser,
				WorkloadTemplateRef: &corev1.TypedObjectReference{
					Kind:      dockyardsv1.WorkloadTemplateKind,
					Name:      "testing",
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
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-super-user",
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

		time.Sleep(time.Second)

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", "test-super-user"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("workloadName", "test-super-user")

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.DeleteClusterWorkload(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-user",
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

		time.Sleep(time.Second)

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", "test-user"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("workloadName", "test-user")

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.DeleteClusterWorkload(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-reader",
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

		time.Sleep(time.Second)

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "workloads", "test-reader"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("clusterName", cluster.Name)
		r.SetPathValue("workloadName", "test-reader")

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.DeleteClusterWorkload(w, r.Clone(ctx))

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
		r.SetPathValue("workloadName", "test-non-existing")

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.DeleteClusterWorkload(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status code %d, got %d", http.StatusNotFound, statusCode)
		}
	})
}
