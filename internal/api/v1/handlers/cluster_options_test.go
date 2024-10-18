package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetClusterOptions(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		expected types.Options
	}{
		{
			name: "test simple",
			lists: []client.ObjectList{
				&dockyardsv1.ReleaseList{
					Items: []dockyardsv1.Release{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								Annotations: map[string]string{
									dockyardsv1.AnnotationDefaultRelease: "true",
								},
							},
							Spec: dockyardsv1.ReleaseSpec{
								Type: dockyardsv1.ReleaseTypeKubernetes,
							},
							Status: dockyardsv1.ReleaseStatus{
								Versions: []string{
									"v1.2.3",
								},
							},
						},
					},
				},
			},
			expected: types.Options{
				Version: []string{
					"v1.2.3",
				},
			},
		},
		{
			name: "test storage role feature",
			lists: []client.ObjectList{
				&dockyardsv1.ReleaseList{
					Items: []dockyardsv1.Release{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								Annotations: map[string]string{
									dockyardsv1.AnnotationDefaultRelease: "true",
								},
							},
							Spec: dockyardsv1.ReleaseSpec{
								Type: dockyardsv1.ReleaseTypeKubernetes,
							},
							Status: dockyardsv1.ReleaseStatus{
								Versions: []string{
									"v1.2.3",
								},
							},
						},
					},
				},
				&dockyardsv1.FeatureList{
					Items: []dockyardsv1.Feature{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      string(featurenames.FeatureStorageRole),
								Namespace: "testing",
							},
						},
					},
				},
			},
			expected: types.Options{
				StorageResourceTypes: &[]string{
					dockyardsv1.StorageResourceTypeHostPath,
				},
				Version: []string{
					"v1.2.3",
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
				Client:    fakeClient,
				namespace: "testing",
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/v1/cluster-options", nil)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.GetClusterOptions(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual types.Options
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
