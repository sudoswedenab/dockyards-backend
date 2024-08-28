package handlers

import (
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

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetApps(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		expected []v1.App
	}{
		{
			name:     "test empty",
			expected: []v1.App{},
		},
		{
			name: "test single",
			lists: []client.ObjectList{
				&v1alpha1.AppList{
					Items: []v1alpha1.App{
						{

							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "86ea7a7c-2c77-49a8-9af2-a36be89aa031",
							},
						},
					},
				},
			},
			expected: []v1.App{
				{
					ID:   "86ea7a7c-2c77-49a8-9af2-a36be89aa031",
					Name: "test",
				},
			},
		},
		{
			name: "test multiple",
			lists: []client.ObjectList{
				&v1alpha1.AppList{
					Items: []v1alpha1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test1",
								UID:  "7a8991b6-0fc8-450b-b97b-d39becc24d89",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test2",
								UID:  "3f09378e-c762-4725-9c28-443055297e75",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test3",
								UID:  "3f72e332-2148-44f3-9266-9f4793c5cf7f",
							},
							Spec: v1alpha1.AppSpec{
								Description: "description",
								Icon:        "http://localhost/icon.png",
							},
						},
					},
				},
			},
			expected: []v1.App{
				{
					ID:   "7a8991b6-0fc8-450b-b97b-d39becc24d89",
					Name: "test1",
				},
				{
					ID:   "3f09378e-c762-4725-9c28-443055297e75",
					Name: "test2",
				},
				{
					ID:          "3f72e332-2148-44f3-9266-9f4793c5cf7f",
					Name:        "test3",
					Description: ptr.To("description"),
					Icon:        ptr.To("http://localhost/icon.png"),
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				Client: fakeClient,
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/v1/apps", nil)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.GetApps(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual []v1.App
			err = json.Unmarshal(body, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body json: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestGetApp(t *testing.T) {
	tt := []struct {
		name     string
		appID    string
		lists    []client.ObjectList
		expected v1.App
	}{
		{
			name:  "test single",
			appID: "7d5fcf7d-e7aa-43da-83e7-700ffc37748e",
			lists: []client.ObjectList{
				&v1alpha1.AppList{
					Items: []v1alpha1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "7d5fcf7d-e7aa-43da-83e7-700ffc37748e",
							},
							Spec: v1alpha1.AppSpec{
								Steps: []v1alpha1.AppStep{
									{
										Name: "step",
										Options: []v1alpha1.AppOption{
											{
												DisplayName: "Helm Chart",
												JSONPointer: "/helm_chart",
												Default:     "test",
											},
											{
												DisplayName: "Helm Repository",
												JSONPointer: "/helm_repository",
												Default:     "http://localhost/chart-repository",
											},
											{
												DisplayName: "Helm Version",
												JSONPointer: "/helm_version",
												Default:     "1.2.3",
											},
											{
												DisplayName: "Helm Ingress Enabled",
												JSONPointer: "/helm_values/ingress/enabled",
												Default:     "true",
												Type:        "boolean",
											},
											{
												DisplayName: "Helm Ingress Host",
												JSONPointer: "/helm_values/ingress/host",
												Default:     "test",
											},
											{
												JSONPointer: "/helm_values/annotations/test",
												Default:     "yes",
												Hidden:      true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: v1.App{
				ID:   "7d5fcf7d-e7aa-43da-83e7-700ffc37748e",
				Name: "test",
				AppSteps: &[]v1.AppStep{
					{
						Name: "step",
						StepOptions: &[]v1.StepOption{
							{
								DisplayName: ptr.To("Helm Chart"),
								JSONPointer: ptr.To("/helm_chart"),
								Default:     ptr.To("test"),
							},
							{
								DisplayName: ptr.To("Helm Repository"),
								JSONPointer: ptr.To("/helm_repository"),
								Default:     ptr.To("http://localhost/chart-repository"),
							},
							{
								DisplayName: ptr.To("Helm Version"),
								JSONPointer: ptr.To("/helm_version"),
								Default:     ptr.To("1.2.3"),
							},
							{
								DisplayName: ptr.To("Helm Ingress Enabled"),
								JSONPointer: ptr.To("/helm_values/ingress/enabled"),
								Default:     ptr.To("true"),
								Type:        ptr.To("boolean"),
							},
							{
								DisplayName: ptr.To("Helm Ingress Host"),
								JSONPointer: ptr.To("/helm_values/ingress/host"),
								Default:     ptr.To("test"),
							},
							{
								JSONPointer: ptr.To("/helm_values/annotations/test"),
								Default:     ptr.To("yes"),
								Hidden:      ptr.To(true),
							},
						},
					},
				},
			},
		},
		{
			name:  "test tags",
			appID: "e4c48f89-dcaf-48e6-b27e-04c34fb651d7",
			lists: []client.ObjectList{
				&v1alpha1.AppList{
					Items: []v1alpha1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "e4c48f89-dcaf-48e6-b27e-04c34fb651d7",
							},
							Spec: v1alpha1.AppSpec{
								Steps: []v1alpha1.AppStep{
									{
										Name: "testing",
										Options: []v1alpha1.AppOption{
											{
												DisplayName: "Ingress Enabled",
												Default:     "false",
												Type:        "boolean",
												Toggle: []string{
													"annotations",
													"tls",
												},
											},
											{
												DisplayName: "TLS Hosts",
												Default:     "test",
												Type:        "hostname",
												Tags: []string{
													"tls",
												},
											},
											{
												DisplayName: "Cluster Issuer",
												Default:     "issuer",
												Tags: []string{
													"annotations",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: v1.App{
				ID:   "e4c48f89-dcaf-48e6-b27e-04c34fb651d7",
				Name: "test",
				AppSteps: &[]v1.AppStep{
					{
						Name: "testing",
						StepOptions: &[]v1.StepOption{
							{
								DisplayName: ptr.To("Ingress Enabled"),
								Default:     ptr.To("false"),
								Type:        ptr.To("boolean"),
								Toggle: &[]string{
									"annotations",
									"tls",
								},
							},
							{
								DisplayName: ptr.To("TLS Hosts"),
								Default:     ptr.To("test"),
								Type:        ptr.To("hostname"),
								Tags: &[]string{
									"tls",
								},
							},
							{
								DisplayName: ptr.To("Cluster Issuer"),
								Default:     ptr.To("issuer"),
								Tags: &[]string{
									"annotations",
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
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.App{}, index.UIDIndexKey, index.UIDIndexer).Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/apps", tc.appID),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("appID", tc.appID)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.GetApp(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.App
			err = json.Unmarshal(body, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body json: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestGetAppErrors(t *testing.T) {
}
