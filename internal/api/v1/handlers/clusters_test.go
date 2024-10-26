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
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateCluster(t *testing.T) {
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

	clusterTemplate := dockyardsv1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    testEnvironment.GetDockyardsNamespace(),
			Annotations: map[string]string{
				dockyardsv1.AnnotationDefaultTemplate: "true",
			},
		},
		Spec: dockyardsv1.ClusterTemplateSpec{
			NodePoolTemplates: []dockyardsv1.NodePoolTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controlplane",
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas:      ptr.To(int32(3)),
						ControlPlane:  true,
						DedicatedRole: true,
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("2"),
							corev1.ResourceMemory:  resource.MustParse("4096M"),
							corev1.ResourceStorage: resource.MustParse("100G"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker",
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas: ptr.To(int32(2)),
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("4"),
							corev1.ResourceMemory:  resource.MustParse("8192M"),
							corev1.ResourceStorage: resource.MustParse("100G"),
						},
					},
				},
			},
		},
	}

	err = c.Create(ctx, &clusterTemplate)
	if err != nil {
		t.Fatal(err)
	}

	release := dockyardsv1.Release{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    testEnvironment.GetDockyardsNamespace(),
			Annotations: map[string]string{
				dockyardsv1.AnnotationDefaultRelease: "true",
			},
		},
		Spec: dockyardsv1.ReleaseSpec{
			Type: dockyardsv1.ReleaseTypeKubernetes,
		},
	}

	err = c.Create(ctx, &release)
	if err != nil {
		t.Fatal(err)
	}

	patch := client.MergeFrom(release.DeepCopy())

	release.Status.LatestVersion = "v1.2.3"

	err = c.Status().Patch(ctx, &release, patch)
	if err != nil {
		t.Fatal(err)
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test default as super user", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-super-user",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		expectedCluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterOptions.Name,
				Namespace: organization.Status.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         dockyardsv1.GroupVersion.String(),
						Kind:               dockyardsv1.OrganizationKind,
						Name:               organization.Name,
						UID:                organization.UID,
						BlockOwnerDeletion: ptr.To(true),
					},
				},
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: release.Status.LatestVersion,
			},
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, client.ObjectKeyFromObject(&expectedCluster), &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actualCluster.Spec, expectedCluster.Spec) {
			t.Errorf("diff: %s", cmp.Diff(expectedCluster.Spec, actualCluster.Spec))
		}
	})

	t.Run("test default as user", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-user",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}
	})

	t.Run("test default as reader", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-reader",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test allocate internal ip", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name:               "test",
			AllocateInternalIP: ptr.To(true),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		objectKey := client.ObjectKey{
			Name:      clusterOptions.Name,
			Namespace: organization.Status.NamespaceRef.Name,
		}

		var actual dockyardsv1.Cluster
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.ClusterSpec{
			AllocateInternalIP: true,
			Version:            "v1.2.3",
		}

		if !cmp.Equal(actual.Spec, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec))
		}
	})

	t.Run("test cluster template", func(t *testing.T) {
		clusterTemplate := dockyardsv1.ClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    testEnvironment.GetDockyardsNamespace(),
			},
			Spec: dockyardsv1.ClusterTemplateSpec{
				NodePoolTemplates: []dockyardsv1.NodePoolTemplate{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "controlplane",
						},
						Spec: dockyardsv1.NodePoolSpec{
							Replicas:      ptr.To(int32(1)),
							ControlPlane:  true,
							DedicatedRole: true,
							Resources: corev1.ResourceList{
								corev1.ResourceCPU:     resource.MustParse("2"),
								corev1.ResourceMemory:  resource.MustParse("3Mi"),
								corev1.ResourceStorage: resource.MustParse("4G"),
							},
						},
					},
				},
			},
		}

		err := c.Create(ctx, &clusterTemplate)
		if err != nil {
			t.Fatal(err)
		}

		clusterOptions := types.ClusterOptions{
			Name:            "test-cluster-template",
			ClusterTemplate: ptr.To(clusterTemplate.Name),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		expectedCluster := dockyardsv1.ClusterSpec{
			Version: "v1.2.3",
		}

		objectKey := client.ObjectKey{
			Name:      clusterOptions.Name,
			Namespace: organization.Status.NamespaceRef.Name,
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, objectKey, &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actualCluster.Spec, expectedCluster) {
			t.Errorf("diff: %s", cmp.Diff(expectedCluster, actualCluster.Spec))
		}

		/*expected: []client.Object{
			&dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-cluster-template-controlplane",
					Namespace:       "testing",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         dockyardsv1.GroupVersion.String(),
							Kind:               dockyardsv1.ClusterKind,
							Name:               "test-cluster-template",
							BlockOwnerDeletion: ptr.To(true),
						},
					},
				},
				Spec: dockyardsv1.NodePoolSpec{
					Replicas:      ptr.To(int32(1)),
					ControlPlane:  true,
					DedicatedRole: true,
					Resources: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("2"),
						corev1.ResourceMemory:  resource.MustParse("3Mi"),
						corev1.ResourceStorage: resource.MustParse("4G"),
					},
				},
			},
		},*/
	})

	t.Run("test invalid organization", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-invalid-organization",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", "invalid-organization", "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", "invalid-organization")

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test invalid name", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "InvalidClusterName",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid node pool name", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-node-pool-name",
			NodePoolOptions: ptr.To([]types.NodePoolOptions{
				{
					Name: "InvalidNodePoolName",
				},
			}),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test existing name", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Status.NamespaceRef.Name,
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		clusterOptions := types.ClusterOptions{
			Name: cluster.Name,
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, statusCode)
		}
	})

	t.Run("test high quantity", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-high-quantity",
			NodePoolOptions: ptr.To([]types.NodePoolOptions{
				{
					Name:     "test",
					Quantity: 123,
				},
			}),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test custom release", func(t *testing.T) {
		release := dockyardsv1.Release{}

		clusterTemplate := dockyardsv1.ClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "custom-release-",
				Namespace:    testEnvironment.GetDockyardsNamespace(),
			},
			Spec: dockyardsv1.ClusterTemplateSpec{
				NodePoolTemplates: []dockyardsv1.NodePoolTemplate{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "controlplane",
						},
						Spec: dockyardsv1.NodePoolSpec{
							Replicas:      ptr.To(int32(1)),
							ControlPlane:  true,
							DedicatedRole: true,
							Resources: corev1.ResourceList{
								corev1.ResourceCPU:     resource.MustParse("2"),
								corev1.ResourceMemory:  resource.MustParse("3Mi"),
								corev1.ResourceStorage: resource.MustParse("4G"),
							},
							ReleaseRef: &corev1.TypedObjectReference{
								Kind:      dockyardsv1.ReleaseKind,
								Name:      release.Name,
								Namespace: ptr.To(release.Namespace),
							},
						},
					},
				},
			},
		}

		err := c.Create(ctx, &clusterTemplate)
		if err != nil {
			t.Fatal(err)
		}

		clusterOptions := types.ClusterOptions{
			Name:            "test-custom-release",
			ClusterTemplate: ptr.To(clusterTemplate.Name),
			Version:         ptr.To("v2.3.4"),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		expectedCluster := dockyardsv1.ClusterSpec{
			Version: "v2.3.4",
		}

		objectKey := client.ObjectKey{
			Name:      clusterOptions.Name,
			Namespace: organization.Status.NamespaceRef.Name,
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, objectKey, &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actualCluster.Spec, expectedCluster) {
			t.Errorf("diff: %s", cmp.Diff(expectedCluster, actualCluster.Spec))
		}

		/*expected: []client.Object{
			&dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-custom-release",
					Namespace:       "testing",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         dockyardsv1.GroupVersion.String(),
							Kind:               dockyardsv1.OrganizationKind,
							Name:               "test",
							BlockOwnerDeletion: ptr.To(true),
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v2.3.4",
				},
			},
			&dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-custom-release-controlplane",
					Namespace:       "testing",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         dockyardsv1.GroupVersion.String(),
							Kind:               dockyardsv1.ClusterKind,
							Name:               "test-custom-release",
							BlockOwnerDeletion: ptr.To(true),
						},
					},
				},
				Spec: dockyardsv1.NodePoolSpec{
					Replicas:      ptr.To(int32(1)),
					ControlPlane:  true,
					DedicatedRole: true,
					Resources: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("2"),
						corev1.ResourceMemory:  resource.MustParse("3Mi"),
						corev1.ResourceStorage: resource.MustParse("4G"),
					},
					ReleaseRef: &corev1.TypedObjectReference{
						Kind:      dockyardsv1.ReleaseKind,
						Name:      "custom",
						Namespace: ptr.To("dockyards-testing"),
					},
				},
			},
		}*/
	})

	t.Run("test storage resources", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-storage-resources",
			NodePoolOptions: &[]types.NodePoolOptions{
				{
					Name:     "worker",
					Quantity: 3,
					DiskSize: ptr.To("4G"),
					RAMSize:  ptr.To("3Mi"),
					CPUCount: ptr.To(2),
					StorageResources: &[]types.StorageResource{
						{
							Name:     "test",
							Quantity: "123",
							Type:     ptr.To("HostPath"),
						},
					},
				},
			},
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		/*
			expected: []client.Object{
				&dockyardsv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-storage-resources",
						Namespace:       "testing",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         dockyardsv1.GroupVersion.String(),
								Kind:               dockyardsv1.OrganizationKind,
								Name:               "test",
								BlockOwnerDeletion: ptr.To(true),
							},
						},
					},
					Spec: dockyardsv1.ClusterSpec{
						Version: "v1.2.3",
					},
				},
				&dockyardsv1.NodePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-storage-resources-worker",
						Namespace:       "testing",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         dockyardsv1.GroupVersion.String(),
								Kind:               dockyardsv1.ClusterKind,
								Name:               "test-storage-resources",
								BlockOwnerDeletion: ptr.To(true),
							},
						},
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas: ptr.To(int32(3)),
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("2"),
							corev1.ResourceMemory:  resource.MustParse("3Mi"),
							corev1.ResourceStorage: resource.MustParse("4G"),
						},
						StorageResources: []dockyardsv1.NodePoolStorageResource{
							{
								Name:     "test",
								Quantity: resource.MustParse("123"),
								Type:     dockyardsv1.StorageResourceTypeHostPath,
							},
						},
					},
				},
			}*/

	})

	t.Run("test duration", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name:     "test-duration",
			Duration: ptr.To("15m"),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.CreateCluster(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}
		/*expected: []client.Object{
			&dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-duration",
					Namespace:       "testing",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         dockyardsv1.GroupVersion.String(),
							Kind:               dockyardsv1.OrganizationKind,
							Name:               "test",
							BlockOwnerDeletion: ptr.To(true),
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.2.3",
					Duration: &metav1.Duration{
						Duration: time.Minute * 15,
					},
				},
			},
		}*/
	})
}

func TestDeleteCluster(t *testing.T) {
	tt := []struct {
		name      string
		clusterID string
		sub       string
		lists     []client.ObjectList
	}{
		{
			name:      "test simple",
			clusterID: "43257a3d-426d-458b-af8e-4aefad29d442",
			sub:       "7994b631-399a-41e6-9c6c-200391f8f87d",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
								UID:  "10659cb0-fce0-4155-b8c6-4b6b825b6da9",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "7994b631-399a-41e6-9c6c-200391f8f87d",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "43257a3d-426d-458b-af8e-4aefad29d442",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "10659cb0-fce0-4155-b8c6-4b6b825b6da9",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.DeleteCluster(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusAccepted {
				t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
			}
		})
	}

}

func TestDeleteClusterErrors(t *testing.T) {
	tt := []struct {
		name      string
		clusterID string
		sub       string
		lists     []client.ObjectList
		expected  int
	}{
		{
			name:     "test empty",
			expected: http.StatusBadRequest,
		},
		{
			name:      "test invalid cluster",
			clusterID: "cluster-123",
			sub:       "f5cf8f91-2b38-4bf4-bb52-d4d4f79f42c3",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "f5cf8f91-2b38-4bf4-bb52-d4d4f79f42c3",
									},
								},
							},
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:      "test invalid organization membership",
			clusterID: "cluster-123",
			sub:       "8ce52ca1-1931-49a1-8ddf-62bf3870a4bf",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "0b8f6617-eba7-4360-b73a-11dac2286a40",
									},
								},
							},
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.DeleteCluster(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestGetCluster(t *testing.T) {
	now := metav1.Now()

	tt := []struct {
		name      string
		clusterID string
		sub       string
		lists     []client.ObjectList
		expected  types.Cluster
	}{
		{
			name:      "test simple",
			clusterID: "26836276-22c6-41bc-bb40-78cdf141e302",
			sub:       "f235721e-8e34-4b57-a6aa-8f6d31162a41",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
								UID:  "fca014c1-a753-4867-9ed3-9d59a4cb89d3",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "f235721e-8e34-4b57-a6aa-8f6d31162a41",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "test",
								Namespace:         "testing",
								UID:               "26836276-22c6-41bc-bb40-78cdf141e302",
								CreationTimestamp: now,
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "fca014c1-a753-4867-9ed3-9d59a4cb89d3",
									},
								},
							},
							Status: dockyardsv1.ClusterStatus{
								Conditions: []metav1.Condition{
									{
										Type:    dockyardsv1.ReadyCondition,
										Status:  metav1.ConditionTrue,
										Reason:  "testing",
										Message: "active",
									},
								},
								Version: "v1.2.3",
							},
						},
					},
				},
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-pool",
								UID:  "14edb8e7-b76a-48c7-bfd8-81588d243c33",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "test",
										UID:        "26836276-22c6-41bc-bb40-78cdf141e302",
									},
								},
							},
						},
					},
				},
			},
			expected: types.Cluster{
				Name:         "test",
				ID:           "26836276-22c6-41bc-bb40-78cdf141e302",
				Organization: "test-org",
				CreatedAt:    now.Time.Truncate(time.Second),
				NodePools: []types.NodePool{
					{
						ID:        "14edb8e7-b76a-48c7-bfd8-81588d243c33",
						Name:      "test-pool",
						ClusterID: "26836276-22c6-41bc-bb40-78cdf141e302",
					},
				},
				State:   "active",
				Version: "v1.2.3",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).
				WithIndex(&dockyardsv1.NodePool{}, index.OwnerReferencesField, index.ByOwnerReferences).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetCluster(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual types.Cluster
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestGetClusterErrors(t *testing.T) {
	tt := []struct {
		name      string
		clusterID string
		sub       string
		lists     []client.ObjectList
		expected  int
	}{
		{
			name:     "test empty",
			expected: http.StatusBadRequest,
		},
		{
			name:      "test invalid cluster",
			clusterID: "9aaa7968-e06e-4b71-98b4-0acdd37b957f",
			expected:  http.StatusUnauthorized,
		},
		{
			name:      "test invalid membership",
			clusterID: "f8d06eb3-e43d-4057-b200-97062c6d96cc",
			sub:       "f6f6531f-ab6c-4237-b1cb-76133674465f",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-org",
								Namespace: "test",
								UID:       "aa1e5599-1cf4-4b50-9020-79b4492a5545",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "afb03005-d51d-4387-9857-83125ff505d5",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "f8d06eb3-e43d-4057-b200-97062c6d96cc",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "aa1e5599-1cf4-4b50-9020-79b4492a5545",
									},
								},
							},
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetCluster(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestGetClusters(t *testing.T) {
	now := metav1.Now()

	tt := []struct {
		name     string
		sub      string
		lists    []client.ObjectList
		expected []types.Cluster
	}{
		{
			name: "test single cluster",
			sub:  "7945098c-e2ef-451b-8cbf-d9674bddd031",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "e7f0fc59-5cae-4208-a97b-a8e46c999821",
								Name: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "7945098c-e2ef-451b-8cbf-d9674bddd031",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:               "072d27ef-3675-48bf-8a47-748f1ae6d3ec",
								Name:              "cluster1",
								Namespace:         "testing",
								CreationTimestamp: now,
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "e7f0fc59-5cae-4208-a97b-a8e46c999821",
									},
								},
							},
							Status: dockyardsv1.ClusterStatus{
								Version: "v1.2.3",
							},
						},
					},
				},
			},
			expected: []types.Cluster{
				{
					ID:           "072d27ef-3675-48bf-8a47-748f1ae6d3ec",
					Name:         "cluster1",
					Organization: "test",
					CreatedAt:    now.Time.Truncate(time.Second),
					Version:      "v1.2.3",
				},
			},
		},
		{
			name: "test cluster without organization",
			sub:  "9142a815-562b-4b1e-b5fd-2163845cced5",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "391aa7e8-999d-4f41-9815-29bd39e94c41",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "9142a815-562b-4b1e-b5fd-2163845cced5",
									},
								},
							},
						},
					},
				},
			},
			expected: []types.Cluster{},
		},
		{
			name: "test cluster with internal ip allocation",
			sub:  "c05034fd-1a84-4723-bfc0-b565ed925ebf",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "a4b99d4b-7abe-4e2b-96f7-fd75063755f2",
								Name: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "c05034fd-1a84-4723-bfc0-b565ed925ebf",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:               "8ff763a6-876b-485e-871f-e000ff27e53c",
								Name:              "internal-ip-allocation",
								Namespace:         "testing",
								CreationTimestamp: now,
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "a4b99d4b-7abe-4e2b-96f7-fd75063755f2",
									},
								},
							},
							Spec: dockyardsv1.ClusterSpec{
								AllocateInternalIP: true,
							},
							Status: dockyardsv1.ClusterStatus{
								Version: "v1.2.3",
							},
						},
					},
				},
			},
			expected: []types.Cluster{
				{

					ID:                 "8ff763a6-876b-485e-871f-e000ff27e53c",
					Name:               "internal-ip-allocation",
					Organization:       "test",
					CreatedAt:          now.Time.Truncate(time.Second),
					Version:            "v1.2.3",
					AllocateInternalIP: ptr.To(true),
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Organization{}, index.MemberReferencesField, index.ByMemberReferences).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetClusters(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual []types.Cluster
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
