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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetDeployment(t *testing.T) {
	now := time.Now()

	tt := []struct {
		name         string
		deploymentID string
		lists        []client.ObjectList
		expected     types.Deployment
	}{
		{
			name:         "test container image",
			deploymentID: "9f72e4e6-412c-47a9-b3e8-8704e129db57",
			lists: []client.ObjectList{
				&dockyardsv1.ContainerImageDeploymentList{
					Items: []dockyardsv1.ContainerImageDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "d175e475-f22e-470a-883b-07915401c88b",
							},
							Spec: dockyardsv1.ContainerImageDeploymentSpec{
								Image: "docker.io/library/nginx:latest",
								Port:  1234,
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "9f72e4e6-412c-47a9-b3e8-8704e129db57",
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceUser,
								DeploymentRefs: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &dockyardsv1.GroupVersion.Group,
										Kind:     dockyardsv1.ContainerImageDeploymentKind,
										Name:     "test",
									},
								},
							},
						},
					},
				},
			},
			expected: types.Deployment{
				ID:             "9f72e4e6-412c-47a9-b3e8-8704e129db57",
				Provenience:    ptr.To(dockyardsv1.ProvenienceUser),
				Name:           ptr.To("test"),
				ContainerImage: ptr.To("docker.io/library/nginx:latest"),
				Port:           ptr.To(1234),
			},
		},
		{
			name:         "test helm chart with values",
			deploymentID: "5621d3b0-0d4e-4265-9d92-56a580bcdd74",
			lists: []client.ObjectList{
				&dockyardsv1.HelmDeploymentList{
					Items: []dockyardsv1.HelmDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "27636906-3857-4445-8b9f-7ab30306a27a",
							},
							Spec: dockyardsv1.HelmDeploymentSpec{
								Chart:      "test-chart",
								Repository: "http://localhost",
								Version:    "v1.2.3",
								Values: &apiextensionsv1.JSON{
									Raw: []byte("{\"testing\":true,\"count\":123}"),
								},
							},
						}},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "5621d3b0-0d4e-4265-9d92-56a580bcdd74",
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceDockyards,
								DeploymentRefs: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &dockyardsv1.GroupVersion.Group,
										Kind:     dockyardsv1.HelmDeploymentKind,
										Name:     "test",
									},
								},
							},
						},
					},
				},
			},
			expected: types.Deployment{
				ID:             "5621d3b0-0d4e-4265-9d92-56a580bcdd74",
				Provenience:    ptr.To(dockyardsv1.ProvenienceDockyards),
				Name:           ptr.To("test"),
				HelmChart:      ptr.To("test-chart"),
				HelmRepository: ptr.To("http://localhost"),
				HelmVersion:    ptr.To("v1.2.3"),
				HelmValues: ptr.To(map[string]any{
					"testing": true,
					"count":   float64(123),
				}),
			},
		},
		{
			name:         "test deployment with single status",
			deploymentID: "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
			lists: []client.ObjectList{
				&dockyardsv1.ContainerImageDeploymentList{
					Items: []dockyardsv1.ContainerImageDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "166da931-27c5-4044-bedd-ecf4dd01d6ee",
							},
							Spec: dockyardsv1.ContainerImageDeploymentSpec{
								Image: "test",
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceUser,
								DeploymentRefs: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &dockyardsv1.GroupVersion.Group,
										Kind:     dockyardsv1.ContainerImageDeploymentKind,
										Name:     "test",
									},
								},
							},
							Status: dockyardsv1.DeploymentStatus{
								Conditions: []metav1.Condition{
									{
										LastTransitionTime: metav1.Time{Time: now},
										Type:               dockyardsv1.ReadyCondition,
										Status:             metav1.ConditionFalse,
										Reason:             "testing",
										Message:            "testing",
									},
								},
							},
						},
					},
				},
			},
			expected: types.Deployment{
				ID:             "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
				Provenience:    ptr.To(dockyardsv1.ProvenienceUser),
				Name:           ptr.To("test"),
				ContainerImage: ptr.To("test"),
				Status: &types.DeploymentStatus{
					CreatedAt: now.Truncate(time.Second),
					State:     ptr.To("testing"),
					Health:    ptr.To(types.DeploymentStatusHealthWarning),
				},
			},
		},
		{
			name:         "test kustomize",
			deploymentID: "c12c2313-662c-4895-86c2-49837c845086",
			lists: []client.ObjectList{
				&dockyardsv1.KustomizeDeploymentList{
					Items: []dockyardsv1.KustomizeDeployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "15776a26-a83e-4354-bc1d-df9ae6865e45",
							},
							Spec: dockyardsv1.KustomizeDeploymentSpec{
								Kustomize: map[string][]byte{
									"kustomization.yaml": []byte("kustomize"),
									"test.yaml":          []byte("hello"),
								},
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "c12c2313-662c-4895-86c2-49837c845086",
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceUser,
								DeploymentRefs: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &dockyardsv1.GroupVersion.Group,
										Kind:     dockyardsv1.KustomizeDeploymentKind,
										Name:     "test",
									},
								},
							},
						},
					},
				},
			},
			expected: types.Deployment{
				ID:          "c12c2313-662c-4895-86c2-49837c845086",
				Provenience: ptr.To(dockyardsv1.ProvenienceUser),
				Name:        ptr.To("test"),
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("kustomize"),
					"test.yaml":          []byte("hello"),
				}),
			},
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
				WithIndex(&dockyardsv1.Deployment{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/deployments", tc.deploymentID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("deploymentID", tc.deploymentID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.GetDeployment(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual types.Deployment
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
		deploymentID string
		lists        []client.ObjectList
		expected     int
	}{
		{
			name:         "test missing",
			deploymentID: "c1e4b45e-cfe3-4fc7-a73a-2a3908524271",
			expected:     http.StatusUnauthorized,
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
				WithIndex(&dockyardsv1.Deployment{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/deployments", tc.deploymentID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("deploymentID", tc.deploymentID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.GetDeployment(w, r.Clone(ctx))

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
		clusterID string
		lists     []client.ObjectList
		expected  []types.Deployment
	}{
		{
			name:      "test single deployment",
			clusterID: "9746d1c6-01d3-4d24-b552-7888d5119a7e",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "b4715218-c084-4c1e-b59f-29a0c5848681",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "05658865-b26f-485c-a1bf-b008552aa7ce",
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
								UID:       "9746d1c6-01d3-4d24-b552-7888d5119a7e",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "b4715218-c084-4c1e-b59f-29a0c5848681",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-test",
								Namespace: "testing",
								UID:       "115590c5-c5f5-48d3-95b4-5fd6a1d3e77f",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "test",
										UID:        "9746d1c6-01d3-4d24-b552-7888d5119a7e",
									},
								},
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceDockyards,
							},
						},
					},
				},
			},
			expected: []types.Deployment{
				{
					ID:          "115590c5-c5f5-48d3-95b4-5fd6a1d3e77f",
					Provenience: ptr.To(dockyardsv1.ProvenienceDockyards),
					Name:        ptr.To("test"),
					ClusterID:   "9746d1c6-01d3-4d24-b552-7888d5119a7e",
				},
			},
		},
		{
			name:      "test multiple deployments",
			clusterID: "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "d7efe04a-517b-4726-b84b-6cec573c3601",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "1ef182bf-2b0a-46ec-ae6d-13c0c62cd1c9",
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
								Name:      "cluster-123",
								Namespace: "testing",
								UID:       "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
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
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "d7efe04a-517b-4726-b84b-6cec573c3601",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-123-test1",
								Namespace: "testing",
								UID:       "9f5be117-7a87-4b14-8788-42b595cd7679",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "cluster-123",
										UID:        "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
									},
								},
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceDockyards,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-234-test2",
								Namespace: "testing",
								UID:       "d40c37d3-7465-4bc6-bfbf-19669f05a16a",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "cluster-234",
										UID:        "8bf6e7fa-2492-4e8a-9597-0041fc49d3ee",
									},
								},
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceDockyards,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "cluster-123-test3",
								Namespace: "testing",
								UID:       "a7743bee-d4cc-4342-b7bd-d149fa26f38f",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "cluster-123",
										UID:        "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
									},
								},
							},
							Spec: dockyardsv1.DeploymentSpec{
								Provenience: dockyardsv1.ProvenienceUser,
							},
						},
					},
				},
			},
			expected: []types.Deployment{
				{
					ID:          "9f5be117-7a87-4b14-8788-42b595cd7679",
					Provenience: ptr.To(dockyardsv1.ProvenienceDockyards),
					Name:        ptr.To("test1"),
					ClusterID:   "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
				},
				{
					ID:          "a7743bee-d4cc-4342-b7bd-d149fa26f38f",
					Provenience: ptr.To(dockyardsv1.ProvenienceUser),
					Name:        ptr.To("test3"),
					ClusterID:   "f7fbef40-3ee7-45f3-af1d-5a810b074ef1",
				},
			},
		},
		{
			name:      "test cluster without deployments",
			clusterID: "d1359b49-9190-45f0-b586-b5240fea847c",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "bf876395-0282-4e5f-8eec-48db0ddfff12",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.OrganizationMemberReference{
									{
										TypedLocalObjectReference: corev1.TypedLocalObjectReference{
											Name: "test",
										},
										UID: "465cef30-6793-422a-9b50-bd081353ea22",
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
								UID:       "d1359b49-9190-45f0-b586-b5240fea847c",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.OrganizationKind,
										Name:       "test",
										UID:        "bf876395-0282-4e5f-8eec-48db0ddfff12",
									},
								},
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test1",
								Namespace: "testing",
								UID:       "b6cf669a-601f-4543-9a3c-d65da2d176d2",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
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
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
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
										APIVersion: dockyardsv1.GroupVersion.String(),
										Kind:       dockyardsv1.ClusterKind,
										Name:       "cluster-345",
										UID:        "fcf10d81-9e9b-4792-ab61-3cb668497529",
									},
								},
							},
						},
					},
				},
			},
			expected: []types.Deployment{},
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
				WithIndex(&dockyardsv1.Cluster{}, index.UIDField, index.ByUID).
				WithIndex(&dockyardsv1.Deployment{}, index.OwnerReferencesField, index.ByOwnerReferences).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters/", tc.clusterID, "deployments"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.GetClusterDeployments(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			var actual []types.Deployment
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
		deploymentID string
		lists        []client.ObjectList
	}{
		{
			name:         "test single",
			deploymentID: "33de82a0-4133-45dc-b319-ab6a8a1daebc",
			lists: []client.ObjectList{
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
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
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Deployment{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/deployments", tc.deploymentID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

			r.SetPathValue("deploymentID", tc.deploymentID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.DeleteDeployment(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusNoContent {
				t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
			}

			var deployment dockyardsv1.Deployment
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
		deployment types.Deployment
		lists      []client.ObjectList
		expected   types.Deployment
	}{
		{
			name:      "test helm",
			clusterID: "b75471ce-5967-4633-a54d-270c7e7c7f26",
			deployment: types.Deployment{
				HelmChart:      ptr.To("test"),
				HelmRepository: ptr.To("http://localhost"),
				HelmVersion:    ptr.To("v1.2.3"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			expected: types.Deployment{
				Type:           types.DeploymentTypeHelm,
				ClusterID:      "b75471ce-5967-4633-a54d-270c7e7c7f26",
				Name:           ptr.To("test"),
				Namespace:      ptr.To("test"),
				HelmChart:      ptr.To("test"),
				HelmRepository: ptr.To("http://localhost"),
				HelmVersion:    ptr.To("v1.2.3"),
			},
		},
		{
			name:      "test container image",
			clusterID: "fb858a98-ac5f-44a8-9a51-f839077c1a93",
			deployment: types.Deployment{
				ContainerImage: ptr.To("test"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			expected: types.Deployment{
				Type:           types.DeploymentTypeContainerImage,
				ClusterID:      "fb858a98-ac5f-44a8-9a51-f839077c1a93",
				Name:           ptr.To("test"),
				Namespace:      ptr.To("test"),
				ContainerImage: ptr.To("docker.io/library/test"),
			},
		},
		{
			name:      "test kustomize",
			clusterID: "4c924548-e827-4005-b335-d6880e23a9e1",
			deployment: types.Deployment{
				Name: ptr.To("test"),
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("testing"),
				}),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			expected: types.Deployment{
				Type:      types.DeploymentTypeKustomize,
				ClusterID: "4c924548-e827-4005-b335-d6880e23a9e1",
				Name:      ptr.To("test"),
				Namespace: ptr.To("test"),
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("testing"),
				}),
			},
		},
		{
			name:      "test container image with credential",
			clusterID: "07ce5009-c89e-458a-b2b5-07390f6e6d28",
			deployment: types.Deployment{
				Name:           ptr.To("test"),
				ContainerImage: ptr.To("test"),
				CredentialName: ptr.To("test-123"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
								Name:      "test-123",
								Namespace: "testing",
								UID:       "74e1819c-8b20-4187-b464-17f9d2c229a8",
							},
							Type: dockyardsv1.SecretTypeCredential,
						},
					},
				},
			},
			expected: types.Deployment{
				Type:           types.DeploymentTypeContainerImage,
				ClusterID:      "07ce5009-c89e-458a-b2b5-07390f6e6d28",
				Name:           ptr.To("test"),
				Namespace:      ptr.To("test"),
				ContainerImage: ptr.To("docker.io/library/test"),
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
				WithIndex(&corev1.Secret{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			b, err := json.Marshal(tc.deployment)
			if err != nil {
				t.Fatalf("%s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.PostClusterDeployments(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}

			b, err = io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual types.Deployment
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
		deployment types.Deployment
		lists      []client.ObjectList
		expected   int
	}{
		{
			name:      "test invalid name",
			clusterID: "3ef173a1-4929-4f68-902d-c88110d0920d",
			deployment: types.Deployment{
				Name: ptr.To("InvalidName"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			deployment: types.Deployment{
				Name:           ptr.To("test"),
				ContainerImage: ptr.To("http://localhost:1234/my-image"),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test name already in-use",
			clusterID: "e4f31f20-8cdd-421b-9fb6-633b84f9b9e9",
			deployment: types.Deployment{
				Name: ptr.To("test"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "e4f31f20-8cdd-421b-9fb6-633b84f9b9e9",
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
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
		{
			name:      "test invalid credential name",
			clusterID: "1a62c42f-a021-4fef-92d6-664a0be27e27",
			deployment: types.Deployment{
				Name:           ptr.To("test"),
				CredentialName: ptr.To("invalid-credential"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "1a62c42f-a021-4fef-92d6-664a0be27e27",
							},
						},
					},
				},
			},
			expected: http.StatusForbidden,
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
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			b, err := json.Marshal(tc.deployment)
			if err != nil {
				t.Fatalf("unexpected error marshalling test deployment: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.PostClusterDeployments(w, r.Clone(ctx))

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
		deployment types.Deployment
		lists      []client.ObjectList
	}{
		{
			name:      "test container image",
			clusterID: "da6c6ca1-5a6d-4ebd-b96a-e7c7140654b6",
			deployment: types.Deployment{
				ContainerImage: ptr.To("test"),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			deployment: types.Deployment{
				ContainerImage: ptr.To("nginx:l.2"),
				Port:           ptr.To(1234),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			deployment: types.Deployment{
				Name: ptr.To("kustomize"),
				Kustomize: ptr.To(map[string][]byte{
					"kustomization.yaml": []byte("hello"),
				}),
			},
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			b, err := json.Marshal(tc.deployment)
			if err != nil {
				t.Fatalf("error marshalling deployment: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("clusterID", tc.clusterID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.PostClusterDeployments(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}
		})
	}
}
