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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetDeployment(t *testing.T) {
	now := time.Now()

	tt := []struct {
		name         string
		deploymentId string
		lists        []client.ObjectList
		expected     v1.Deployment
	}{
		{
			name:         "test container image",
			deploymentId: "9f72e4e6-412c-47a9-b3e8-8704e129db57",
			lists: []client.ObjectList{
				&v1alpha1.ContainerImageDeploymentList{
					Items: []v1alpha1.ContainerImageDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "d175e475-f22e-470a-883b-07915401c88b",
							},
							Spec: v1alpha1.ContainerImageDeploymentSpec{
								Image: "docker.io/library/nginx:latest",
								Port:  1234,
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "9f72e4e6-412c-47a9-b3e8-8704e129db57",
							},
							Spec: v1alpha1.DeploymentSpec{
								DeploymentRef: v1alpha1.DeploymentReference{
									APIVersion: v1alpha1.GroupVersion.String(),
									Kind:       v1alpha1.ContainerImageDeploymentKind,
									Name:       "test",
									UID:        "d175e475-f22e-470a-883b-07915401c88b",
								},
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Id:             "9f72e4e6-412c-47a9-b3e8-8704e129db57",
				Name:           util.Ptr("test"),
				ContainerImage: util.Ptr("docker.io/library/nginx:latest"),
				Port:           util.Ptr(1234),
			},
		},
		{
			name:         "test helm chart with values",
			deploymentId: "5621d3b0-0d4e-4265-9d92-56a580bcdd74",
			lists: []client.ObjectList{
				&v1alpha1.HelmDeploymentList{
					Items: []v1alpha1.HelmDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "27636906-3857-4445-8b9f-7ab30306a27a",
							},
							Spec: v1alpha1.HelmDeploymentSpec{
								Chart:      "test-chart",
								Repository: "http://localhost",
								Version:    "v1.2.3",
								Values: &apiextensionsv1.JSON{
									Raw: []byte("{\"testing\":true,\"count\":123}"),
								},
							},
						}},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "5621d3b0-0d4e-4265-9d92-56a580bcdd74",
							},
							Spec: v1alpha1.DeploymentSpec{
								DeploymentRef: v1alpha1.DeploymentReference{
									APIVersion: v1alpha1.GroupVersion.String(),
									Kind:       v1alpha1.HelmDeploymentKind,
									Name:       "test",
									UID:        "27636906-3857-4445-8b9f-7ab30306a27a",
								},
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Id:             "5621d3b0-0d4e-4265-9d92-56a580bcdd74",
				Name:           util.Ptr("test"),
				HelmChart:      util.Ptr("test-chart"),
				HelmRepository: util.Ptr("http://localhost"),
				HelmVersion:    util.Ptr("v1.2.3"),
				HelmValues: util.Ptr(map[string]any{
					"testing": true,
					"count":   float64(123),
				}),
			},
		},
		{
			name:         "test deployment with single status",
			deploymentId: "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
			lists: []client.ObjectList{
				&v1alpha1.ContainerImageDeploymentList{
					Items: []v1alpha1.ContainerImageDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "166da931-27c5-4044-bedd-ecf4dd01d6ee",
							},
							Spec: v1alpha1.ContainerImageDeploymentSpec{
								Image: "test",
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
							},
							Spec: v1alpha1.DeploymentSpec{
								DeploymentRef: v1alpha1.DeploymentReference{
									APIVersion: v1alpha1.GroupVersion.String(),
									Kind:       v1alpha1.ContainerImageDeploymentKind,
									Name:       "test",
								},
							},
							Status: v1alpha1.DeploymentStatus{
								Conditions: []metav1.Condition{
									{
										LastTransitionTime: metav1.Time{Time: now},
										Type:               v1alpha1.ReadyCondition,
										Status:             metav1.ConditionFalse,
										Reason:             v1alpha1.DeploymentReadyReason,
										Message:            "testing",
									},
								},
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Id:             "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
				Name:           util.Ptr("test"),
				ContainerImage: util.Ptr("test"),
				Status: &v1.DeploymentStatus{
					CreatedAt: now.Truncate(time.Second),
					State:     util.Ptr("testing"),
					Health:    util.Ptr(v1.DeploymentStatusHealthWarning),
				},
			},
		},
		{
			name:         "test kustomize",
			deploymentId: "c12c2313-662c-4895-86c2-49837c845086",
			lists: []client.ObjectList{
				&v1alpha1.KustomizeDeploymentList{
					Items: []v1alpha1.KustomizeDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "15776a26-a83e-4354-bc1d-df9ae6865e45",
							},
							Spec: v1alpha1.KustomizeDeploymentSpec{
								Kustomize: map[string][]byte{
									"kustomization.yaml": []byte("kustomize"),
									"test.yaml":          []byte("hello"),
								},
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "c12c2313-662c-4895-86c2-49837c845086",
							},
							Spec: v1alpha1.DeploymentSpec{
								DeploymentRef: v1alpha1.DeploymentReference{
									APIVersion: v1alpha1.GroupVersion.String(),
									Kind:       v1alpha1.KustomizeDeploymentKind,
									Name:       "test",
								},
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Id:   "c12c2313-662c-4895-86c2-49837c845086",
				Name: util.Ptr("test"),
				Kustomize: util.Ptr(map[string][]byte{
					"kustomization.yaml": []byte("kustomize"),
					"test.yaml":          []byte("hello"),
				}),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&v1alpha1.Deployment{}, index.UIDIndexKey, index.UIDIndexer).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/deployments/:deploymentID", h.GetDeployment)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/deployments", tc.deploymentId),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.Deployment
			err = json.Unmarshal(b, &actual)

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("difference between actual and expected: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestGetDeploymentErrors(t *testing.T) {
	tt := []struct {
		name         string
		deploymentId string
		lists        []client.ObjectList
		expected     int
	}{
		{
			name:         "test missing",
			deploymentId: "c1e4b45e-cfe3-4fc7-a73a-2a3908524271",
			expected:     http.StatusUnauthorized,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.Deployment{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/deployments/:deploymentID", h.GetDeployment)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/deployments", tc.deploymentId),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestGetClusterDeployments(t *testing.T) {
	tt := []struct {
		name      string
		clusterId string
		lists     []client.ObjectList
		expected  []v1.Deployment
	}{
		{
			name:      "test single deployment",
			clusterId: "9746d1c6-01d3-4d24-b552-7888d5119a7e",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "b4715218-c084-4c1e-b59f-29a0c5848681",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "05658865-b26f-485c-a1bf-b008552aa7ce",
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
								UID:       "9746d1c6-01d3-4d24-b552-7888d5119a7e",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "b4715218-c084-4c1e-b59f-29a0c5848681",
									},
								},
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-test",
								Namespace: "testing",
								UID:       "115590c5-c5f5-48d3-95b4-5fd6a1d3e77f",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "test",
										UID:        "9746d1c6-01d3-4d24-b552-7888d5119a7e",
									},
								},
							},
						},
					},
				},
			},
			expected: []v1.Deployment{
				{
					Id:        "115590c5-c5f5-48d3-95b4-5fd6a1d3e77f",
					Name:      util.Ptr("test"),
					ClusterId: "9746d1c6-01d3-4d24-b552-7888d5119a7e",
				},
			},
		},
		{
			name:      "test multiple deployments",
			clusterId: "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "d7efe04a-517b-4726-b84b-6cec573c3601",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "1ef182bf-2b0a-46ec-ae6d-13c0c62cd1c9",
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
								Name:      "cluster-123",
								Namespace: "testing",
								UID:       "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "d7efe04a-517b-4726-b84b-6cec573c3601",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-234",
								Namespace: "testing",
								UID:       "8bf6e7fa-2492-4e8a-9597-0041fc49d3ee",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "d7efe04a-517b-4726-b84b-6cec573c3601",
									},
								},
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-123-test1",
								Namespace: "testing",
								UID:       "9f5be117-7a87-4b14-8788-42b595cd7679",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "cluster-123",
										UID:        "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-234-test2",
								Namespace: "testing",
								UID:       "d40c37d3-7465-4bc6-bfbf-19669f05a16a",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "cluster-234",
										UID:        "8bf6e7fa-2492-4e8a-9597-0041fc49d3ee",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-123-test3",
								Namespace: "testing",
								UID:       "a7743bee-d4cc-4342-b7bd-d149fa26f38f",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "cluster-123",
										UID:        "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
									},
								},
							},
						},
					},
				},
			},
			expected: []v1.Deployment{
				{
					Id:        "9f5be117-7a87-4b14-8788-42b595cd7679",
					Name:      util.Ptr("test1"),
					ClusterId: "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
				},
				{
					Id:        "a7743bee-d4cc-4342-b7bd-d149fa26f38f",
					Name:      util.Ptr("test3"),
					ClusterId: "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
				},
			},
		},
		{
			name:      "test cluster without deployments",
			clusterId: "d1359b49-9190-45f0-b586-b5240fea847c",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "bf876395-0282-4e5f-8eec-48db0ddfff12",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "465cef30-6793-422a-9b50-bd081353ea22",
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
								UID:       "d1359b49-9190-45f0-b586-b5240fea847c",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.OrganizationKind,
										Name:       "test",
										UID:        "bf876395-0282-4e5f-8eec-48db0ddfff12",
									},
								},
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test1",
								Namespace: "testing",
								UID:       "b6cf669a-601f-4543-9a3c-d65da2d176d2",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "cluster-123",
										UID:        "6b446452-2522-45db-aee3-4c3df0acc181",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test2",
								Namespace: "testing",
								UID:       "1748bcf1-92c7-482e-a07c-a808701b2d84",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "cluster-234",
										UID:        "8bf6e7fa-2492-4e8a-9597-0041fc49d3ee",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test3",
								Namespace: "testing",
								UID:       "fd9786ad-6722-4ac4-9e18-6a128472eb60",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: v1alpha1.GroupVersion.String(),
										Kind:       v1alpha1.ClusterKind,
										Name:       "cluster-345",
										UID:        "fcf10d81-9e9b-4792-ab61-3cb668497529",
									},
								},
							},
						},
					},
				},
			},
			expected: []v1.Deployment{},
		},
		/*{
			name:      "test with deployment status",
			clusterId: "e96f28f3-a2f9-426c-8e9d-9ffdba4b8581",
			deployments: []v1.Deployment{
				{
					Id:        "2a0d2f6d-e3b1-4021-84cd-5c47918dc99e",
					ClusterId: "e96f28f3-a2f9-426c-8e9d-9ffdba4b8581",
				},
			},
			deploymentStatuses: []v1.DeploymentStatus{
				{
					Id:           "dce9a76b-1a68-4d5d-bcea-fef85a265882",
					DeploymentId: "fe9c90d4-6c0d-4038-8099-e4075bc1484b",
					State:        util.Ptr("testing"),
				},
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "e96f28f3-a2f9-426c-8e9d-9ffdba4b8581",
							},
						},
					},
				},
			},
			expected: []v1.Deployment{
				{
					Id:        "2a0d2f6d-e3b1-4021-84cd-5c47918dc99e",
					ClusterId: "e96f28f3-a2f9-426c-8e9d-9ffdba4b8581",
				},
			},
		},*/
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&v1alpha1.Cluster{}, index.UIDIndexKey, index.UIDIndexer).
				WithIndex(&v1alpha1.Deployment{}, index.OwnerRefsIndexKey, index.OwnerRefsIndexer).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/clusters/:clusterID/deployments", h.GetClusterDeployments)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/clusters/", tc.clusterId, "deployments"),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			var actual []v1.Deployment
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("expected no error unmarshalling reponse, got %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))

			}
		})
	}
}

func TestDeleteDeployment(t *testing.T) {
	tt := []struct {
		name         string
		deploymentId string
		lists        []client.ObjectList
	}{
		{
			name:         "test single",
			deploymentId: "33de82a0-4133-45dc-b319-ab6a8a1daebc",
			lists: []client.ObjectList{
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-123",
								Namespace: "testing",
								UID:       "33de82a0-4133-45dc-b319-ab6a8a1daebc",
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
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.Deployment{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.DELETE("/v1/deployments/:deploymentID", h.DeleteDeployment)

			c.Request = &http.Request{
				Method: http.MethodDelete,
				URL: &url.URL{
					Path: path.Join("/v1/deployments", tc.deploymentId),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusNoContent {
				t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
			}

			var deployment v1alpha1.Deployment
			err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: "test-123", Namespace: "testing"}, &deployment)
			if !apierrors.IsNotFound(err) {
				t.Errorf("expected is not found error, got '%s'", err)
			}
		})
	}
}

func TestPostClusterDeployments(t *testing.T) {
	tt := []struct {
		name       string
		clusterID  string
		deployment v1.Deployment
		lists      []client.ObjectList
		expected   v1.Deployment
	}{
		{
			name:      "test helm",
			clusterID: "b75471ce-5967-4633-a54d-270c7e7c7f26",
			deployment: v1.Deployment{
				HelmChart:      util.Ptr("test"),
				HelmRepository: util.Ptr("http://localhost"),
				HelmVersion:    util.Ptr("v1.2.3"),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "b75471ce-5967-4633-a54d-270c7e7c7f26",
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Type:           v1.DeploymentTypeHelm,
				ClusterId:      "b75471ce-5967-4633-a54d-270c7e7c7f26",
				Name:           util.Ptr("test"),
				Namespace:      util.Ptr("test"),
				HelmChart:      util.Ptr("test"),
				HelmRepository: util.Ptr("http://localhost"),
				HelmVersion:    util.Ptr("v1.2.3"),
			},
		},
		{
			name:      "test container image",
			clusterID: "fb858a98-ac5f-44a8-9a51-f839077c1a93",
			deployment: v1.Deployment{
				ContainerImage: util.Ptr("test"),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "fb858a98-ac5f-44a8-9a51-f839077c1a93",
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Type:           v1.DeploymentTypeContainerImage,
				ClusterId:      "fb858a98-ac5f-44a8-9a51-f839077c1a93",
				Name:           util.Ptr("test"),
				Namespace:      util.Ptr("test"),
				ContainerImage: util.Ptr("docker.io/library/test"),
			},
		},
		{
			name:      "test kustomize",
			clusterID: "4c924548-e827-4005-b335-d6880e23a9e1",
			deployment: v1.Deployment{
				Name: util.Ptr("test"),
				Kustomize: util.Ptr(map[string][]byte{
					"kustomization.yaml": []byte("testing"),
				}),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "4c924548-e827-4005-b335-d6880e23a9e1",
							},
						},
					},
				},
			},
			expected: v1.Deployment{
				Type:      v1.DeploymentTypeKustomize,
				ClusterId: "4c924548-e827-4005-b335-d6880e23a9e1",
				Name:      util.Ptr("test"),
				Namespace: util.Ptr("test"),
				Kustomize: util.Ptr(map[string][]byte{
					"kustomization.yaml": []byte("testing"),
				}),
			},
		},
		{
			name:      "test container image with credential",
			clusterID: "07ce5009-c89e-458a-b2b5-07390f6e6d28",
			deployment: v1.Deployment{
				Name:           util.Ptr("test"),
				ContainerImage: util.Ptr("test"),
				CredentialId:   util.Ptr("74e1819c-8b20-4187-b464-17f9d2c229a8"),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "07ce5009-c89e-458a-b2b5-07390f6e6d28",
							},
						},
					},
				},
				&corev1.SecretList{
					Items: []corev1.Secret{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "74e1819c-8b20-4187-b464-17f9d2c229a8",
							},
							Type: DockyardsSecretTypeCredential,
						},
					},
				},
			},
			expected: v1.Deployment{
				Type:           v1.DeploymentTypeContainerImage,
				ClusterId:      "07ce5009-c89e-458a-b2b5-07390f6e6d28",
				Name:           util.Ptr("test"),
				Namespace:      util.Ptr("test"),
				ContainerImage: util.Ptr("docker.io/library/test"),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&v1alpha1.Cluster{}, index.UIDIndexKey, index.UIDIndexer).
				WithIndex(&corev1.Secret{}, index.UIDIndexKey, index.UIDIndexer).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.POST("/v1/clusters/:clusterID/deployments", h.PostClusterDeployments)

			b, err := json.Marshal(tc.deployment)
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			req, err := http.NewRequest(http.MethodPost, u.String(), buf)

			r.ServeHTTP(w, req)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}

			b, err = io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.Deployment
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body json: %s", err)
			}

			ignoreTypes := []any{uuid.UUID{}, time.Time{}}
			if !cmp.Equal(actual, tc.expected, cmpopts.IgnoreTypes(ignoreTypes...)) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual, cmpopts.IgnoreTypes(ignoreTypes...)))
			}
		})
	}
}

func TestPostClusterDeploymentsErrors(t *testing.T) {
	tt := []struct {
		name       string
		clusterID  string
		deployment v1.Deployment
		lists      []client.ObjectList
		expected   int
	}{
		{
			name:      "test invalid name",
			clusterID: "3ef173a1-4929-4f68-902d-c88110d0920d",
			deployment: v1.Deployment{
				Name: util.Ptr("InvalidName"),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "3ef173a1-4929-4f68-902d-c88110d0920d",
							},
						},
					},
				},
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test invalid container image",
			clusterID: "58705bd3-fe06-4c67-8651-a61294bcff8e",
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "58705bd3-fe06-4c67-8651-a61294bcff8e",
							},
						},
					},
				},
			},
			deployment: v1.Deployment{
				Name:           util.Ptr("test"),
				ContainerImage: util.Ptr("http://localhost:1234/my-image"),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test name already in-use",
			clusterID: "e4f31f20-8cdd-421b-9fb6-633b84f9b9e9",
			deployment: v1.Deployment{
				Name: util.Ptr("test"),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "e4f31f20-8cdd-421b-9fb6-633b84f9b9e9",
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-test",
								Namespace: "testing",
								UID:       "0b6530a2-9c8b-4397-9183-aaf86c1f9af5",
							},
						},
					},
				},
			},
			expected: http.StatusConflict,
		},
	}

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
			_, r := gin.CreateTestContext(w)
			r.POST("/v1/clusters/:clusterID/deployments", h.PostClusterDeployments)

			b, err := json.Marshal(tc.deployment)
			if err != nil {
				t.Fatalf("unexpected error marshalling test deployment: %s", err)
			}
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			req, err := http.NewRequest(http.MethodPost, u.String(), buf)

			r.ServeHTTP(w, req)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestPostClusterDeploymentsContainerImage(t *testing.T) {
	tt := []struct {
		name       string
		clusterID  string
		deployment v1.Deployment
		lists      []client.ObjectList
	}{
		{
			name:      "test container image",
			clusterID: "da6c6ca1-5a6d-4ebd-b96a-e7c7140654b6",
			deployment: v1.Deployment{
				ContainerImage: util.Ptr("test"),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "da6c6ca1-5a6d-4ebd-b96a-e7c7140654b6",
							},
						},
					},
				},
			},
		},
		{
			name:      "test port",
			clusterID: "faeb3f05-1d92-4b7c-adaa-127c15ee6296",
			deployment: v1.Deployment{
				ContainerImage: util.Ptr("nginx:l.2"),
				Port:           util.Ptr(1234),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "faeb3f05-1d92-4b7c-adaa-127c15ee6296",
							},
						},
					},
				},
			},
		},
		{
			name:      "test kustomize",
			clusterID: "c32fe438-b956-414d-90ad-40d37143c2f0",
			deployment: v1.Deployment{
				Name: util.Ptr("kustomize"),
				Kustomize: util.Ptr(map[string][]byte{
					"kustomization.yaml": []byte("hello"),
				}),
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "c32fe438-b956-414d-90ad-40d37143c2f0",
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
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&v1alpha1.Cluster{}, index.UIDIndexKey, index.UIDIndexer).
				Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.POST("/v1/clusters/:clusterID/deployments", h.PostClusterDeployments)

			b, err := json.Marshal(tc.deployment)
			if err != nil {
				t.Fatalf("error marshalling deployment: %s", err)
			}

			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			req, err := http.NewRequest(http.MethodPost, u.String(), buf)

			r.ServeHTTP(w, req)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}
		})
	}
}
