package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetNodePool(t *testing.T) {
	tt := []struct {
		name       string
		nodePoolID string
		sub        string
		lists      []client.ObjectList
		expected   v1.NodePool
	}{
		{
			name:       "test single node",
			nodePoolID: "0c386e60-e759-426f-b39d-36588547fac0",
			sub:        "74eab97f-f635-4ec9-99b1-40f37fde690d",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "845e9322-8dbe-4eed-bda2-5efe2b54dc71",
								Name: "test",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "74eab97f-f635-4ec9-99b1-40f37fde690d",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "test-123",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "test-123",
								UID:       "31f38488-c0df-48fe-89f8-e94a6c8c3258",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "845e9322-8dbe-4eed-bda2-5efe2b54dc71",
									},
								},
							},
						},
					},
				},
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-pool",
								Namespace: "test-123",
								UID:       "0c386e60-e759-426f-b39d-36588547fac0",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "31f38488-c0df-48fe-89f8-e94a6c8c3258",
									},
								},
							},
						},
					},
				},
				&v1alpha1.NodeList{
					Items: []v1alpha1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-pool-1",
								Namespace: "test-123",
								UID:       "55310c2b-589b-4044-8fce-8544ce0faf6c",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.NodePoolKind,
										Name:       "test-pool",
										UID:        "0c386e60-e759-426f-b39d-36588547fac0",
									},
								},
							},
						},
					},
				},
			},
			expected: v1.NodePool{
				Id:        "0c386e60-e759-426f-b39d-36588547fac0",
				ClusterId: "31f38488-c0df-48fe-89f8-e94a6c8c3258",
				Name:      "test-pool",
				Nodes: []v1.Node{
					{
						Id:   "55310c2b-589b-4044-8fce-8544ce0faf6c",
						Name: "test-pool-1",
					},
				},
			},
		},
		{
			name:       "test complex node",
			nodePoolID: "4386082b-cabe-4235-b6be-a857706ed6f4",
			sub:        "ab46bc94-446c-4448-bc13-94b25e61bd37",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "8be3ef94-5742-4fb7-9302-28d23da783da",
								Name: "test",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "ab46bc94-446c-4448-bc13-94b25e61bd37",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "3fac0683-34bf-4f8a-908b-28db92cf20a0",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "8be3ef94-5742-4fb7-9302-28d23da783da",
									},
								},
							},
						},
					},
				},
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-complex",
								Namespace: "testing",
								UID:       "4386082b-cabe-4235-b6be-a857706ed6f4",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "3fac0683-34bf-4f8a-908b-28db92cf20a0",
									},
								},
							},
							Spec: v1alpha1.NodePoolSpec{
								ControlPlane: true,
								Replicas:     util.Ptr(int32(3)),
								Resources: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("123Gi"),
								},
							},
							Status: v1alpha1.NodePoolStatus{
								Resources: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("123Gi"),
								},
							},
						},
					},
				},
				&v1alpha1.NodeList{
					Items: []v1alpha1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-complex-1",
								Namespace: "testing",
								UID:       "55310c2b-589b-4044-8fce-8544ce0faf6c",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.NodePoolKind,
										Name:       "test-complex",
										UID:        "4386082b-cabe-4235-b6be-a857706ed6f4",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-complex-2",
								Namespace: "testing",
								UID:       "0b5f21f5-aa1e-4286-be18-b172cb162ff4",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.NodePoolKind,
										Name:       "test-complex",
										UID:        "4386082b-cabe-4235-b6be-a857706ed6f4",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-complex-3",
								Namespace: "testing",
								UID:       "e50f48b7-0332-4824-b8ea-139d5a0a5d39",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.NodePoolKind,
										Name:       "test-complex",
										UID:        "4386082b-cabe-4235-b6be-a857706ed6f4",
									},
								},
							},
						},
					},
				},
			},
			expected: v1.NodePool{
				Id:           "4386082b-cabe-4235-b6be-a857706ed6f4",
				ClusterId:    "3fac0683-34bf-4f8a-908b-28db92cf20a0",
				ControlPlane: util.Ptr(true),
				DiskSize:     "123Gi",
				Name:         "test-complex",
				Quantity:     3,
				Nodes: []v1.Node{
					{
						Name: "test-complex-1",
						Id:   "55310c2b-589b-4044-8fce-8544ce0faf6c",
					},
					{
						Name: "test-complex-2",
						Id:   "0b5f21f5-aa1e-4286-be18-b172cb162ff4",
					},
					{
						Name: "test-complex-3",
						Id:   "e50f48b7-0332-4824-b8ea-139d5a0a5d39",
					},
				},
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&v1alpha1.NodePool{}, index.UIDIndexKey, index.UIDIndexer).
				WithIndex(&v1alpha1.Node{}, index.OwnerRefsIndexKey, index.OwnerRefsIndexer).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)
			c.Params = []gin.Param{
				{Key: "nodePoolID", Value: tc.nodePoolID},
			}
			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/node-pools", tc.nodePoolID),
				},
			}

			h.GetNodePool(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual v1.NodePool
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body to json: %s", err)
			}

			if !cmp.Equal(tc.expected, actual) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestGetNodePoolErrors(t *testing.T) {
	tt := []struct {
		name       string
		nodePoolID string
		sub        string
		lists      []client.ObjectList
		expected   int
	}{
		{
			name:       "test invalid node pool",
			nodePoolID: "c46ece0d-33cc-4097-9f99-98471a2a8acb",
			expected:   http.StatusUnauthorized,
		},
		{
			name:       "test invalid cluster",
			nodePoolID: "1864d09d-c68d-4ab0-a47e-24a9eb86235b",
			lists: []client.ObjectList{
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "1864d09d-c68d-4ab0-a47e-24a9eb86235b",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "941cbe7b-341b-40a3-b4b8-645aa0317b1c",
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
			name:       "test invalid organization",
			nodePoolID: "c70e8ac9-d415-4596-9e00-f000cf42f277",
			lists: []client.ObjectList{
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "c70e8ac9-d415-4596-9e00-f000cf42f277",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "484226da-101d-4a7d-9620-f4507cc928c0",
									},
								},
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "484226da-101d-4a7d-9620-f4507cc928c0",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "a7516b46-cbf7-4a3b-a0bc-a7777d233ad2",
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
			name:       "test invalid membership",
			nodePoolID: "b86ebca0-71d1-498a-ab34-6e6b68600af3",
			sub:        "e33dbae7-d222-43be-afc2-23e52654a7d3",
			lists: []client.ObjectList{
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "b86ebca0-71d1-498a-ab34-6e6b68600af3",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "d13e643a-40bc-47d1-acc0-011c4d9c1faf",
									},
								},
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "d13e643a-40bc-47d1-acc0-011c4d9c1faf",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "69bacf70-a70c-4db0-a389-8816e6109a11",
									},
								},
							},
						},
					},
				},
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "69bacf70-a70c-4db0-a389-8816e6109a11",
								Name: "test",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "5125130c-a4af-40b6-8b36-b8be8f4d2fdb",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
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
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)
			c.Params = []gin.Param{
				{Key: "nodePoolID", Value: tc.nodePoolID},
			}

			u := url.URL{
				Path: path.Join("/v1/node-pools", tc.nodePoolID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error creating test request: %s", err)
			}

			h.GetNodePool(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Errorf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestPostClusterNodePools(t *testing.T) {
	tt := []struct {
		name            string
		clusterID       string
		nodePoolOptions v1.NodePoolOptions
		sub             string
		lists           []client.ObjectList
		expected        v1.NodePool
	}{
		{
			name:      "test simple",
			clusterID: "acf90c2f-62ea-4b5d-9636-bf08ed0dcac5",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			sub: "d80ff784-20fe-4bcc-b52f-e57764111c9a",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "3928f445-d53c-4a23-9663-77382a361d17",
								Name: "test-org",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "d80ff784-20fe-4bcc-b52f-e57764111c9a",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "acf90c2f-62ea-4b5d-9636-bf08ed0dcac5",
								Name:      "cluster1",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test-org",
										UID:        "3928f445-d53c-4a23-9663-77382a361d17",
									},
								},
							},
						},
					},
				},
			},
			expected: v1.NodePool{
				ClusterId: "acf90c2f-62ea-4b5d-9636-bf08ed0dcac5",
				Name:      "cluster1-test",
			},
		},
		{
			name:      "test complex",
			clusterID: "b70dc16e-1c52-4861-9932-59d950edcc49",
			nodePoolOptions: v1.NodePoolOptions{
				Name:                       "test2",
				Quantity:                   3,
				LoadBalancer:               util.Ptr(true),
				ControlPlaneComponentsOnly: util.Ptr(true),
				RamSize:                    util.Ptr("1234M"),
				CpuCount:                   util.Ptr(12),
				DiskSize:                   util.Ptr("123Gi"),
			},
			sub: "940b43ee-39d3-4ecb-a6bd-be25245d7eca",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "a86dd064-4fa5-489f-ab29-6f49f92a38eb",
								Name: "test-org",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "940b43ee-39d3-4ecb-a6bd-be25245d7eca",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "b70dc16e-1c52-4861-9932-59d950edcc49",
								Name:      "cluster-123",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test-org",
										UID:        "a86dd064-4fa5-489f-ab29-6f49f92a38eb",
									},
								},
							},
						},
					},
				},
			},
			expected: v1.NodePool{
				Name:                       "cluster-123-test2",
				ClusterId:                  "b70dc16e-1c52-4861-9932-59d950edcc49",
				Quantity:                   3,
				LoadBalancer:               util.Ptr(true),
				ControlPlaneComponentsOnly: util.Ptr(true),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.Cluster{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)
			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}

			b, err := json.Marshal(tc.nodePoolOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test options: %s", err)
			}
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "node-pools"),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), buf)
			if err != nil {
				t.Fatalf("unxepected error creating test request: %s", err)
			}

			h.PostClusterNodePools(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}

			b, err = io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual v1.NodePool
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body to json: %s", err)
			}

			if !cmp.Equal(tc.expected, actual) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}

		})
	}
}

func TestPostClusterNodePoolsErrors(t *testing.T) {
	tt := []struct {
		name            string
		clusterID       string
		nodePoolOptions v1.NodePoolOptions
		sub             string
		lists           []client.ObjectList
		expected        int
	}{
		{
			name:      "test invalid cluster",
			clusterID: "1817bd8b-db70-46ce-bc05-5d99df68b79e",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:      "test invalid name",
			clusterID: "a2e90092-956c-4ac9-8ec7-8d4e757faf25",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "InvalidName",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test conflict name",
			clusterID: "57cd048f-ceff-4d12-a19c-d8edab370d06",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			sub: "df24c8f4-27f3-485a-ae7a-92546b3fb925",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "ae19c385-6254-4d73-a2fa-53c29796ee91",
								Name: "test-org",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "df24c8f4-27f3-485a-ae7a-92546b3fb925",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "57cd048f-ceff-4d12-a19c-d8edab370d06",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test-org",
										UID:        "ae19c385-6254-4d73-a2fa-53c29796ee91",
									},
								},
							},
						},
					},
				},
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "57a48eaa-87ac-4bdc-bd77-541e72c77df3",
								Name:      "test-test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "57cd048f-ceff-4d12-a19c-d8edab370d06",
									},
								},
							},
						},
					},
				},
			},
			expected: http.StatusConflict,
		},
		{
			name:      "test invalid membership",
			clusterID: "0948b965-ea97-4e74-8262-d1b6c1ccc367",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			sub: "44946295-97bc-4c24-8887-69d3f0ca0dad",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "d3570450-a7e1-4201-a16f-b913ad6c7f11",
								Name: "test-org",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "bbc144d1-0f5f-4f8b-8b8b-54d0619395bc",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "0948b965-ea97-4e74-8262-d1b6c1ccc367",
								Name:      "test",
								Namespace: "testing",
							},
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:      "test high quantity",
			clusterID: "3c727788-9cd1-4b74-836b-8b6ff5e58b8b",
			nodePoolOptions: v1.NodePoolOptions{
				Name:     "test",
				Quantity: 50,
			},
			sub: "44946295-97bc-4c24-8887-69d3f0ca0dad",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "d3570450-a7e1-4201-a16f-b913ad6c7f11",
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "bbc144d1-0f5f-4f8b-8b8b-54d0619395bc",
									},
								},
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "3c727788-9cd1-4b74-836b-8b6ff5e58b8b",
								Name:      "test",
								Namespace: "testing",
							},
						},
					},
				},
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.Cluster{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)
			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}

			b, err := json.Marshal(tc.nodePoolOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test options: %s", err)
			}
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "node-pools"),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), buf)
			if err != nil {
				t.Fatalf("unexpected error creating test request: %s", err)
			}

			h.PostClusterNodePools(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestDeleteNodePool(t *testing.T) {
	tt := []struct {
		name       string
		nodePoolID string
		sub        string
		lists      []client.ObjectList
		expected   int
	}{
		{
			name:       "test simple",
			nodePoolID: "18f543f7-ed03-405e-b808-5a562db0105f",
			sub:        "3be51320-a001-4c81-88fd-68e6b0f29a88",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "ed44e536-2387-490d-937f-e415d2246daa",
								Name: "test",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "3be51320-a001-4c81-88fd-68e6b0f29a88",
									},
								},
							},
							Status: v1alpha1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "22a4f9ab-bbdb-465f-8b4a-3c51c5111585",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "ed44e536-2387-490d-937f-e415d2246daa",
									},
								},
							},
						},
					},
				},
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "18f543f7-ed03-405e-b808-5a562db0105f",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "6ee4de08-8834-4e06-95d3-ad3c9f91c68c",
									},
								},
							},
						},
					},
				},
			},
			expected: http.StatusNoContent,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.NodePool{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)
			c.Params = []gin.Param{
				{Key: "nodePoolID", Value: tc.nodePoolID},
			}

			u := url.URL{
				Path: path.Join("/v1/node-pools", tc.nodePoolID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error creating test request: %s", err)
			}

			h.DeleteNodePool(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestDeleteNodePoolErrors(t *testing.T) {
	tt := []struct {
		name       string
		nodePoolID string
		sub        string
		lists      []client.ObjectList
		expected   int
	}{
		{
			name:       "test invalid node pool id",
			nodePoolID: "864ab209-d7ee-41d5-8d1a-d1b424f5fdcc",
			expected:   http.StatusUnauthorized,
		},
		{
			name:       "test invalid cluster",
			nodePoolID: "1eb79767-2d33-4c6a-babf-1ee41a814eb2",
			lists: []client.ObjectList{
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "1eb79767-2d33-4c6a-babf-1ee41a814eb2",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "dd22e117-c846-4405-be6b-62c39220612d",
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
			name:       "test invalid organization",
			nodePoolID: "ccb52a82-a1e8-43b9-9f3f-4d89e1c2649a",
			lists: []client.ObjectList{
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "ccb52a82-a1e8-43b9-9f3f-4d89e1c2649a",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "296f57de-d8b3-45ea-831a-fef90c850ca2",
									},
								},
							},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "296f57de-d8b3-45ea-831a-fef90c850ca2",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "6b396f62-0988-400c-a465-ad1e2b90a570",
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
			name:       "test invalid membership",
			nodePoolID: "6d67527f-accd-439e-a2e9-89d66ea244e8",
			sub:        "3df06ce8-2806-4807-beec-89e7f6199b6e",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "7293a952-c798-4d3e-a998-541ba978d33d",
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: v1alpha1.OrganizationSpec{},
						},
					},
				},
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "6610727b-623c-49d1-a1fe-d45004e65d75",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "7293a952-c798-4d3e-a998-541ba978d33d",
									},
								},
							},
						},
					},
				},
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "6d67527f-accd-439e-a2e9-89d66ea244e8",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "6610727b-623c-49d1-a1fe-d45004e65d75",
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
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.NodePool{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)
			c.Params = []gin.Param{
				{Key: "nodePoolID", Value: tc.nodePoolID},
			}

			u := url.URL{
				Path: path.Join("/v1/node-pools", tc.nodePoolID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error creating test request: %s", err)
			}

			h.DeleteNodePool(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}
