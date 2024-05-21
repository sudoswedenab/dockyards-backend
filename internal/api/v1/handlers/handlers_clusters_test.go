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

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func TestPostOrgClusters(t *testing.T) {
	tt := []struct {
		name             string
		organizationName string
		sub              string
		lists            []client.ObjectList
		clusterOptions   v1.ClusterOptions
		expected         []client.Object
	}{
		{
			name:             "test recommended",
			organizationName: "test-org",
			sub:              "fec813fc-7938-4cb9-ba12-bb28f6b1f5d9",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "fec813fc-7938-4cb9-ba12-bb28f6b1f5d9",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&dockyardsv1.ClusterTemplateList{
					Items: []dockyardsv1.ClusterTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "recommended",
							},
							Spec: dockyardsv1.ClusterTemplateSpec{
								NodePoolTemplates: []dockyardsv1.NodePool{
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "control-plane",
										},
										Spec: dockyardsv1.NodePoolSpec{
											Replicas:      util.Ptr(int32(3)),
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
											Replicas: util.Ptr(int32(2)),
											Resources: corev1.ResourceList{
												corev1.ResourceCPU:     resource.MustParse("4"),
												corev1.ResourceMemory:  resource.MustParse("8192M"),
												corev1.ResourceStorage: resource.MustParse("100G"),
											},
										},
									},
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "load-balancer",
										},
										Spec: dockyardsv1.NodePoolSpec{
											Replicas:      util.Ptr(int32(2)),
											LoadBalancer:  true,
											DedicatedRole: true,
											Resources: corev1.ResourceList{
												corev1.ResourceCPU:     resource.MustParse("2"),
												corev1.ResourceMemory:  resource.MustParse("4096M"),
												corev1.ResourceStorage: resource.MustParse("100G"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test",
			},
			expected: []client.Object{
				&dockyardsv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test",
						Namespace:       "testing",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         dockyardsv1.GroupVersion.String(),
								Kind:               dockyardsv1.OrganizationKind,
								Name:               "test-org",
								BlockOwnerDeletion: util.Ptr(true),
							},
						},
					},
				},
				&dockyardsv1.NodePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-control-plane",
						Namespace:       "testing",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         dockyardsv1.GroupVersion.String(),
								Kind:               dockyardsv1.ClusterKind,
								Name:               "test",
								BlockOwnerDeletion: util.Ptr(true),
							},
						},
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas:      util.Ptr(int32(3)),
						ControlPlane:  true,
						DedicatedRole: true,
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("2"),
							corev1.ResourceMemory:  resource.MustParse("4096M"),
							corev1.ResourceStorage: resource.MustParse("100G"),
						},
					},
				},
				&dockyardsv1.NodePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-load-balancer",
						Namespace:       "testing",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         dockyardsv1.GroupVersion.String(),
								Kind:               dockyardsv1.ClusterKind,
								Name:               "test",
								BlockOwnerDeletion: util.Ptr(true),
							},
						},
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas:      util.Ptr(int32(2)),
						LoadBalancer:  true,
						DedicatedRole: true,
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("2"),
							corev1.ResourceMemory:  resource.MustParse("4096M"),
							corev1.ResourceStorage: resource.MustParse("100G"),
						},
					},
				},
				&dockyardsv1.NodePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-worker",
						Namespace:       "testing",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         dockyardsv1.GroupVersion.String(),
								Kind:               dockyardsv1.ClusterKind,
								Name:               "test",
								BlockOwnerDeletion: util.Ptr(true),
							},
						},
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas: util.Ptr(int32(2)),
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

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			b, err := json.Marshal(tc.clusterOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test cluster options: %s", err)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "org", Value: tc.organizationName},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationName, "clusters"),
			}
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.PostOrgClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}

			for _, e := range tc.expected {
				ctx := context.Background()

				objectKey := client.ObjectKey{
					Name:      e.GetName(),
					Namespace: e.GetNamespace(),
				}
				switch x := e.(type) {
				case *dockyardsv1.Cluster:
					var actual dockyardsv1.Cluster
					err := fakeClient.Get(ctx, objectKey, &actual)
					if err != nil {
						t.Errorf("error getting expected cluster: %s", err)
					}

					if !cmp.Equal(x, &actual) {
						t.Errorf("diff: %s", cmp.Diff(x, &actual))
					}
				case *dockyardsv1.NodePool:
					var actual dockyardsv1.NodePool
					err := fakeClient.Get(ctx, objectKey, &actual)
					if err != nil {
						t.Errorf("error getting expected node pool: %s", err)
					}

					if !cmp.Equal(x, &actual) {
						t.Errorf("diff: %s", cmp.Diff(x, &actual))
					}
				default:
					t.Fatalf("test not supported on group version kind: %s", e.GetObjectKind().GroupVersionKind().String())
				}
			}
		})
	}
}

func TestPostOrgClustersErrors(t *testing.T) {
	tt := []struct {
		name             string
		organizationName string
		sub              string
		lists            []client.ObjectList
		clusterOptions   v1.ClusterOptions
		expected         int
	}{
		{
			name:             "test invalid organization",
			organizationName: "test-org",
			expected:         http.StatusUnauthorized,
		},
		{
			name:             "test invalid cluster name",
			organizationName: "test-org",
			sub:              "82aaf116-666f-4846-9e10-defa79a4df3d",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "82aaf116-666f-4846-9e10-defa79a4df3d",
									},
								},
							},
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "InvalidClusterName",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:             "test invalid node pool name",
			organizationName: "test-org",
			sub:              "e7282b48-f8b6-4042-8f4c-12ec59fe3a87",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "e7282b48-f8b6-4042-8f4c-12ec59fe3a87",
									},
								},
							},
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test-cluster",
				NodePoolOptions: util.Ptr([]v1.NodePoolOptions{
					{
						Name: "InvalidNodePoolName",
					},
				}),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:             "test invalid membership",
			organizationName: "test-org",
			sub:              "62034914-3f46-4c71-810f-14ab985399bc",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "af510e3e-e667-4500-8a73-12f2163f849e",
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
			name:             "test existing cluster name",
			organizationName: "test-org",
			sub:              "c185f9d3-b4c4-4cb1-a567-f786c9ac4a2f",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "c185f9d3-b4c4-4cb1-a567-f786c9ac4a2f",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-cluster",
								Namespace: "testing",
							},
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test-cluster",
			},
			expected: http.StatusConflict,
		},
		{
			name:             "test node pool with high quantity",
			organizationName: "test-org",
			sub:              "7a7d8423-c9e7-46f3-958a-e68fb97b4417",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "7a7d8423-c9e7-46f3-958a-e68fb97b4417",
									},
								},
							},
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test-cluster",
				NodePoolOptions: util.Ptr([]v1.NodePoolOptions{
					{
						Name:     "test",
						Quantity: 123,
					},
				}),
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			b, err := json.Marshal(tc.clusterOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test cluster options: %s", err)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "org", Value: tc.organizationName},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationName, "clusters"),
			}
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.PostOrgClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "7994b631-399a-41e6-9c6c-200391f8f87d",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodDelete, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.DeleteCluster(c)

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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "f5cf8f91-2b38-4bf4-bb52-d4d4f79f42c3",
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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "0b8f6617-eba7-4360-b73a-11dac2286a40",
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

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.DeleteCluster(c)

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
		expected  v1.Cluster
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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "f235721e-8e34-4b57-a6aa-8f6d31162a41",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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
			expected: v1.Cluster{
				Name:         "test",
				Id:           "26836276-22c6-41bc-bb40-78cdf141e302",
				Organization: "test-org",
				CreatedAt:    now.Time.Truncate(time.Second),
				NodePools: []v1.NodePool{
					{
						Id:        "14edb8e7-b76a-48c7-bfd8-81588d243c33",
						Name:      "test-pool",
						ClusterId: "26836276-22c6-41bc-bb40-78cdf141e302",
					},
				},
				State:   "active",
				Version: "v1.2.3",
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).
				WithIndex(&dockyardsv1.NodePool{}, index.OwnerReferencesField, index.ByOwnerReferences).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetCluster(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.Cluster
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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "afb03005-d51d-4387-9857-83125ff505d5",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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

	gin.SetMode(gin.TestMode)

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
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetCluster(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestGetClusterKubeconfig(t *testing.T) {
	tt := []struct {
		name      string
		clusterID string
		sub       string
		lists     []client.ObjectList
		expected  clientcmdapi.Config
	}{
		{
			name:      "test simple",
			clusterID: "8fa24e25-eb7a-428f-a750-e6e8aeee8c93",
			sub:       "9eb06ff5-4299-480c-b957-0b10485d873c",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "7a11a699-fd6f-4d7f-838a-266c1d33a0b8",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "9eb06ff5-4299-480c-b957-0b10485d873c",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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
								UID:       "8fa24e25-eb7a-428f-a750-e6e8aeee8c93",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "7a11a699-fd6f-4d7f-838a-266c1d33a0b8",
									},
								},
							},
							Status: dockyardsv1.ClusterStatus{
								ClusterServiceID: "cluster-123",
							},
						},
					},
				},
				&corev1.SecretList{
					Items: []corev1.Secret{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-kubeconfig",
								Namespace: "testing",
							},
							Data: map[string][]byte{
								"value": []byte("current-context: cluster-123"),
							},
						},
					},
				},
			},
			expected: clientcmdapi.Config{
				CurrentContext: "cluster-123",
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "kubeconfig"),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetClusterKubeconfig(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)

			var actual clientcmdapi.Config
			err = yaml.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body yaml: %s", err)
			}
		})
	}
}

func TestGetClusterKubeconfigErrors(t *testing.T) {
	tt := []struct {
		name      string
		clusterID string
		sub       string
		lists     []client.ObjectList
		expected  int
	}{
		{
			name:     "test empty cluster id",
			expected: http.StatusBadRequest,
		},
		{
			name:      "test invalid cluster id",
			clusterID: "3152f6b4-23fd-4e11-8482-2fb38ddf03bd",
			sub:       "83a44759-56b8-480a-9575-ad0f3519f73a",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "7a11a699-fd6f-4d7f-838a-266c1d33a0b8",
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "83a44759-56b8-480a-9575-ad0f3519f73a",
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
			clusterID: "a6b450d8-4bb0-4aa0-83c3-b30cb55460d2",
			sub:       "ef418237-2fd1-4977-861a-2031094a6ae5",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "7a11a699-fd6f-4d7f-838a-266c1d33a0b8",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "7180bc06-66c1-4494-b53e-e9cc878995a9",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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
								UID:       "a6b450d8-4bb0-4aa0-83c3-b30cb55460d2",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "7a11a699-fd6f-4d7f-838a-266c1d33a0b8",
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
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "kubeconfig"),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetClusterKubeconfig(c)

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
		expected []v1.Cluster
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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "7945098c-e2ef-451b-8cbf-d9674bddd031",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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
			expected: []v1.Cluster{
				{
					Id:           "072d27ef-3675-48bf-8a47-748f1ae6d3ec",
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
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "9142a815-562b-4b1e-b5fd-2163845cced5",
									},
								},
							},
						},
					},
				},
			},
			expected: []v1.Cluster{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&dockyardsv1.Organization{}, index.MemberRefsIndexKey, index.MemberRefsIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("sub", tc.sub)

			u := url.URL{
				Path: path.Join("/v1/clusters"),
			}

			var err error
			c.Request, err = http.NewRequest(http.MethodGet, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual []v1.Cluster
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
