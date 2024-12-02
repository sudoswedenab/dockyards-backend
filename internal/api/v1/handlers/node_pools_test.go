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

func TestGetNodePool(t *testing.T) {
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

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.NodePool{}, index.UIDField, index.ByUID)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Node{}, index.OwnerReferencesField, index.ByOwnerReferences)
	if err != nil {
		t.Fatal(err)
	}

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

	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
		},
	}

	err = c.Create(ctx, &nodePool)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	node := dockyardsv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.NodePoolKind,
					Name:       nodePool.Name,
					UID:        nodePool.UID,
				},
			},
		},
	}

	err = c.Create(ctx, &node)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		expected := types.NodePool{
			ClusterID: string(cluster.UID),
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Nodes: []types.Node{
				{
					ID:   string(node.UID),
					Name: node.Name,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		expected := types.NodePool{
			ClusterID: string(cluster.UID),
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Nodes: []types.Node{
				{
					ID:   string(node.UID),
					Name: node.Name,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		expected := types.NodePool{
			ClusterID: string(cluster.UID),
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Nodes: []types.Node{
				{
					ID:   string(node.UID),
					Name: node.Name,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test non-existing node pool", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/node-pools", "ee346d53-8a20-4bdd-b936-5ddc240153ac"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("nodePoolID", "ee346d53-8a20-4bdd-b936-5ddc240153ac")

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test without membership", func(t *testing.T) {
		otherOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  user.UID,
					},
					{
						Role: dockyardsv1.OrganizationMemberRoleUser,
						UID:  reader.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}

		err = c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		otherCluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       otherOrganization.Name,
						UID:        otherOrganization.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherCluster)
		if err != nil {
			t.Fatal(err)
		}

		otherNodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       otherCluster.Name,
						UID:        otherCluster.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherNodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(otherNodePool.UID)),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("nodePoolID", string(otherNodePool.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}

func TestPostClusterNodePools(t *testing.T) {
	tt := []struct {
		name            string
		clusterID       string
		nodePoolOptions types.NodePoolOptions
		sub             string
		lists           []client.ObjectList
		expected        types.NodePool
	}{
		{
			name:      "test simple",
			clusterID: "acf90c2f-62ea-4b5d-9636-bf08ed0dcac5",
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("test"),
				Quantity: ptr.To(0),
			},
			sub: "d80ff784-20fe-4bcc-b52f-e57764111c9a",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "3928f445-d53c-4a23-9663-77382a361d17",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "d80ff784-20fe-4bcc-b52f-e57764111c9a",
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
								UID:       "acf90c2f-62ea-4b5d-9636-bf08ed0dcac5",
								Name:      "cluster1",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "3928f445-d53c-4a23-9663-77382a361d17",
									},
								},
							},
						},
					},
				},
			},
			expected: types.NodePool{
				ClusterID: "acf90c2f-62ea-4b5d-9636-bf08ed0dcac5",
				Name:      "cluster1-test",
			},
		},
		{
			name:      "test complex",
			clusterID: "b70dc16e-1c52-4861-9932-59d950edcc49",
			nodePoolOptions: types.NodePoolOptions{
				Name:                       ptr.To("test2"),
				Quantity:                   ptr.To(3),
				LoadBalancer:               ptr.To(true),
				ControlPlaneComponentsOnly: ptr.To(true),
				RAMSize:                    ptr.To("1234M"),
				CPUCount:                   ptr.To(12),
				DiskSize:                   ptr.To("123Gi"),
			},
			sub: "940b43ee-39d3-4ecb-a6bd-be25245d7eca",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "a86dd064-4fa5-489f-ab29-6f49f92a38eb",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "940b43ee-39d3-4ecb-a6bd-be25245d7eca",
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
								UID:       "b70dc16e-1c52-4861-9932-59d950edcc49",
								Name:      "cluster-123",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "a86dd064-4fa5-489f-ab29-6f49f92a38eb",
									},
								},
							},
						},
					},
				},
			},
			expected: types.NodePool{
				Name:                       "cluster-123-test2",
				ClusterID:                  "b70dc16e-1c52-4861-9932-59d950edcc49",
				Quantity:                   3,
				LoadBalancer:               ptr.To(true),
				ControlPlaneComponentsOnly: ptr.To(true),
				RAMSize:                    "1234M",
				CPUCount:                   12,
				DiskSize:                   "123Gi",
			},
		},
		{
			name:      "test with storage resource without type",
			clusterID: "h5cb74d7f-274c-4284-bbf4-e2a4b1c7dbc3",
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("storage-resources"),
				Quantity: ptr.To(3),
				StorageResources: &[]types.StorageResource{
					{
						Name:     "test",
						Quantity: "123",
					},
				},
			},
			sub: "e7620e3b-c888-43ce-82b5-b6575bfb4a14",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "8491aaf4-df4b-458b-bc39-4d2d1f2f7d34",
								Name: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "e7620e3b-c888-43ce-82b5-b6575bfb4a14",
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
								UID:       "h5cb74d7f-274c-4284-bbf4-e2a4b1c7dbc3",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "8491aaf4-df4b-458b-bc39-4d2d1f2f7d34",
									},
								},
							},
						},
					},
				},
			},
			expected: types.NodePool{
				Name:      "test-storage-resources",
				ClusterID: "h5cb74d7f-274c-4284-bbf4-e2a4b1c7dbc3",
				Quantity:  3,
				StorageResources: &[]types.StorageResource{
					{
						Name:     "test",
						Quantity: "123",
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
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "node-pools"),
			}

			b, err := json.Marshal(tc.nodePoolOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test options: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.PostClusterNodePools(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}

			b, err = io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual types.NodePool
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
		nodePoolOptions types.NodePoolOptions
		sub             string
		lists           []client.ObjectList
		expected        int
	}{
		{
			name:      "test invalid cluster",
			clusterID: "1817bd8b-db70-46ce-bc05-5d99df68b79e",
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("test"),
				Quantity: ptr.To(0),
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:      "test invalid name",
			clusterID: "a2e90092-956c-4ac9-8ec7-8d4e757faf25",
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("InvalidName"),
				Quantity: ptr.To(0),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test conflict name",
			clusterID: "57cd048f-ceff-4d12-a19c-d8edab370d06",
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("test"),
				Quantity: ptr.To(0),
			},
			sub: "df24c8f4-27f3-485a-ae7a-92546b3fb925",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "ae19c385-6254-4d73-a2fa-53c29796ee91",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "df24c8f4-27f3-485a-ae7a-92546b3fb925",
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
								UID:       "57cd048f-ceff-4d12-a19c-d8edab370d06",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test-org",
										UID:        "ae19c385-6254-4d73-a2fa-53c29796ee91",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "57a48eaa-87ac-4bdc-bd77-541e72c77df3",
								Name:      "test-test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
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
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("test"),
				Quantity: ptr.To(0),
			},
			sub: "44946295-97bc-4c24-8887-69d3f0ca0dad",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "d3570450-a7e1-4201-a16f-b913ad6c7f11",
								Name: "test-org",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "bbc144d1-0f5f-4f8b-8b8b-54d0619395bc",
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
			nodePoolOptions: types.NodePoolOptions{
				Name:     ptr.To("test"),
				Quantity: ptr.To(50),
			},
			sub: "44946295-97bc-4c24-8887-69d3f0ca0dad",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "d3570450-a7e1-4201-a16f-b913ad6c7f11",
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "bbc144d1-0f5f-4f8b-8b8b-54d0619395bc",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
				Path: path.Join("/v1/clusters", tc.clusterID, "node-pools"),
			}

			b, err := json.Marshal(tc.nodePoolOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test options: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.PostClusterNodePools(w, r.Clone(ctx))

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
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:  "ed44e536-2387-490d-937f-e415d2246daa",
								Name: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "3be51320-a001-4c81-88fd-68e6b0f29a88",
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
								UID:       "22a4f9ab-bbdb-465f-8b4a-3c51c5111585",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "ed44e536-2387-490d-937f-e415d2246daa",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "18f543f7-ed03-405e-b808-5a562db0105f",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
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
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.NodePool{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/node-pools", tc.nodePoolID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, nil)

			r.SetPathValue("nodePoolID", tc.nodePoolID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.DeleteNodePool(w, r.Clone(ctx))

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
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "1eb79767-2d33-4c6a-babf-1ee41a814eb2",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
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
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "ccb52a82-a1e8-43b9-9f3f-4d89e1c2649a",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "test",
										UID:        "296f57de-d8b3-45ea-831a-fef90c850ca2",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "296f57de-d8b3-45ea-831a-fef90c850ca2",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
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
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "7293a952-c798-4d3e-a998-541ba978d33d",
								Name:      "test-org",
								Namespace: "test",
							},
							Spec: dockyardsv1.OrganizationSpec{},
						},
					},
				},
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "6610727b-623c-49d1-a1fe-d45004e65d75",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "7293a952-c798-4d3e-a998-541ba978d33d",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								UID:       "6d67527f-accd-439e-a2e9-89d66ea244e8",
								Name:      "test",
								Namespace: "testing",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
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
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.NodePool{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/node-pools", tc.nodePoolID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, nil)

			r.SetPathValue("nodePoolID", tc.nodePoolID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.DeleteNodePool(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestUpdateNodePool(t *testing.T) {
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

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.NodePool{}, index.UIDField, index.ByUID)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-update-node-pool-",
			Namespace:    organization.Status.NamespaceRef.Name,
		},
	}

	err = c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test cpu as super user", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-super-user-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			CPUCount: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.NodePool{
			ClusterID: string(cluster.UID),
			CPUCount:  3,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test cpu as user", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-user-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			CPUCount: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.NodePool{
			ClusterID: string(cluster.UID),
			CPUCount:  3,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test cpu as reader", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-reader-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			CPUCount: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test quantity", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-quantity-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Replicas: ptr.To(int32(1)),
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			Quantity: ptr.To(2),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.NodePool{
			ClusterID: string(cluster.UID),
			CPUCount:  2,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Quantity:  2,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}

		var actualNodePool dockyardsv1.NodePool
		err = c.Get(ctx, client.ObjectKeyFromObject(&nodePool), &actualNodePool)
		if err != nil {
			t.Fatal(err)
		}

		expectedNodePool := dockyardsv1.NodePool{
			ObjectMeta: actualNodePool.ObjectMeta,
			Status:     actualNodePool.Status,
			Spec: dockyardsv1.NodePoolSpec{
				Replicas: ptr.To(int32(2)),
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		if !cmp.Equal(actualNodePool, expectedNodePool) {
			t.Errorf("diff: %s", cmp.Diff(expectedNodePool, actualNodePool))
		}
	})

	t.Run("test storage resources", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-storage-resources-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				StorageResources: []dockyardsv1.NodePoolStorageResource{
					{
						Name: "this-should-be-removed",
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "1",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.NodePool{
			Name: nodePool.Name,
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "1",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
			ID:        string(nodePool.UID),
			ClusterID: string(cluster.UID),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test empty node pool id", func(t *testing.T) {
		update := types.NodePoolOptions{
			Quantity: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", ""),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", "")

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, statusCode)
		}
	})

	t.Run("test change name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-change-name-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			Name: ptr.To("hello"),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test non-existing node pool", func(t *testing.T) {
		update := types.NodePoolOptions{
			Quantity: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", "81d8dc5b-13d3-4250-b59f-34723cf3752c"),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", "81d8dc5b-13d3-4250-b59f-34723cf3752c")

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test invalid storage resource type", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-type-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "100Gi",
					Type:     ptr.To("this-type-does-not-exist"),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, statusCode)
		}
	})

	t.Run("test invalid storage resource quantity", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-type-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "invalid-quantity",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, statusCode)
		}
	})

	t.Run("test invalid disk size", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-disk-size-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			DiskSize: ptr.To("foobar"),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid ram size", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-ram-size-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			RAMSize: ptr.To("foobar"),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test empty storage resource name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-storage-resource-name-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "",
					Quantity: "100Gi",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, statusCode)
		}
	})

	t.Run("test invalid storage resource name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-storage-resource-name-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "<script>giveMeYourCookies()</script>",
					Quantity: "100Gi",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, statusCode)
		}
	})

	t.Run("test invalid cpu count", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-count-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)

		update := types.NodePoolOptions{
			CPUCount: ptr.To(-1),
		}

		u := url.URL{
			Path: path.Join("/v1/node-pools", string(nodePool.UID)),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		r.SetPathValue("nodePoolID", string(nodePool.UID))

		h.UpdateNodePool(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})
}
