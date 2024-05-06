package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetClusterOptions(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		expected v1.Options
	}{
		{
			name: "test simple",
			lists: []client.ObjectList{
				&dockyardsv1.ReleaseList{
					Items: []dockyardsv1.Release{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "supported-kubernetes-releases",
								Namespace: "testing",
							},
							Status: dockyardsv1.ReleaseStatus{
								Versions: []string{
									"v1.2.3",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterTemplateList{
					Items: []dockyardsv1.ClusterTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "recommended",
								Namespace: "testing",
							},
							Spec: dockyardsv1.ClusterTemplateSpec{
								NodePoolTemplates: []dockyardsv1.NodePool{
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "cp",
										},
										Spec: dockyardsv1.NodePoolSpec{
											Replicas:     util.Ptr(int32(3)),
											ControlPlane: true,
											Resources: corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("2"),
												corev1.ResourceMemory: resource.MustParse("4Gi"),
											},
										},
									},
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "lb",
										},
										Spec: dockyardsv1.NodePoolSpec{
											Replicas:      util.Ptr(int32(2)),
											LoadBalancer:  true,
											DedicatedRole: true,
										},
									},
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "w",
										},
										Spec: dockyardsv1.NodePoolSpec{
											Resources: corev1.ResourceList{
												corev1.ResourceStorage: resource.MustParse("123G"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: v1.Options{
				SingleNode: false,
				Version: []string{
					"v1.2.3",
				},
				NodePoolOptions: []v1.NodePoolOptions{
					{
						Name:         "cp",
						Quantity:     3,
						ControlPlane: util.Ptr(true),
						CpuCount:     util.Ptr(2),
						RamSize:      util.Ptr("4Gi"),
					},
					{
						Name:                       "lb",
						Quantity:                   2,
						LoadBalancer:               util.Ptr(true),
						ControlPlaneComponentsOnly: util.Ptr(true),
					},
					{
						Name:     "w",
						Quantity: 1,
						DiskSize: util.Ptr("123G"),
					},
				},
			},
		},
		{
			name: "test binary format",
			lists: []client.ObjectList{
				&dockyardsv1.ReleaseList{
					Items: []dockyardsv1.Release{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "supported-kubernetes-releases",
								Namespace: "testing",
							},
							Status: dockyardsv1.ReleaseStatus{
								Versions: []string{
									"v1.2.3",
								},
							},
						},
					},
				},
				&dockyardsv1.ClusterTemplateList{
					Items: []dockyardsv1.ClusterTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "recommended",
								Namespace: "testing",
							},
							Spec: dockyardsv1.ClusterTemplateSpec{
								NodePoolTemplates: []dockyardsv1.NodePool{
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "cp",
										},
										Spec: dockyardsv1.NodePoolSpec{
											ControlPlane: true,
											Resources: corev1.ResourceList{
												corev1.ResourceMemory: resource.MustParse("4Gi"),
											},
										},
									},
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "w",
										},
										Spec: dockyardsv1.NodePoolSpec{
											Resources: corev1.ResourceList{
												corev1.ResourceStorage: resource.MustParse("123Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: v1.Options{
				SingleNode: false,
				Version: []string{
					"v1.2.3",
				},
				NodePoolOptions: []v1.NodePoolOptions{
					{
						Name:         "cp",
						Quantity:     1,
						ControlPlane: util.Ptr(true),
						RamSize:      util.Ptr("4Gi"),
					},
					{
						Name:     "w",
						Quantity: 1,
						DiskSize: util.Ptr("123Gi"),
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
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
				namespace:        "testing",
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			h.GetClusterOptions(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.Options
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling response body: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("difference between actual and expected: %s", cmp.Diff(tc.expected, actual))
			}

		})
	}
}
