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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
					Id:   "86ea7a7c-2c77-49a8-9af2-a36be89aa031",
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
					Id:   "7a8991b6-0fc8-450b-b97b-d39becc24d89",
					Name: "test1",
				},
				{
					Id:   "3f09378e-c762-4725-9c28-443055297e75",
					Name: "test2",
				},
				{
					Id:          "3f72e332-2148-44f3-9266-9f4793c5cf7f",
					Name:        "test3",
					Description: util.Ptr("description"),
					Icon:        util.Ptr("http://localhost/icon.png"),
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
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			h.GetApps(c)

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
				Id:   "7d5fcf7d-e7aa-43da-83e7-700ffc37748e",
				Name: "test",
				AppSteps: &[]v1.AppStep{
					{
						Name: "step",
						StepOptions: &[]v1.StepOption{
							{
								DisplayName: util.Ptr("Helm Chart"),
								JsonPointer: util.Ptr("/helm_chart"),
								Default:     util.Ptr("test"),
							},
							{
								DisplayName: util.Ptr("Helm Repository"),
								JsonPointer: util.Ptr("/helm_repository"),
								Default:     util.Ptr("http://localhost/chart-repository"),
							},
							{
								DisplayName: util.Ptr("Helm Version"),
								JsonPointer: util.Ptr("/helm_version"),
								Default:     util.Ptr("1.2.3"),
							},
							{
								DisplayName: util.Ptr("Helm Ingress Enabled"),
								JsonPointer: util.Ptr("/helm_values/ingress/enabled"),
								Default:     util.Ptr("true"),
								Type:        util.Ptr("boolean"),
							},
							{
								DisplayName: util.Ptr("Helm Ingress Host"),
								JsonPointer: util.Ptr("/helm_values/ingress/host"),
								Default:     util.Ptr("test"),
							},
							{
								JsonPointer: util.Ptr("/helm_values/annotations/test"),
								Default:     util.Ptr("yes"),
								Hidden:      util.Ptr(true),
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
				Id:   "e4c48f89-dcaf-48e6-b27e-04c34fb651d7",
				Name: "test",
				AppSteps: &[]v1.AppStep{
					{
						Name: "testing",
						StepOptions: &[]v1.StepOption{
							{
								DisplayName: util.Ptr("Ingress Enabled"),
								Default:     util.Ptr("false"),
								Type:        util.Ptr("boolean"),
								Toggle: &[]string{
									"annotations",
									"tls",
								},
							},
							{
								DisplayName: util.Ptr("TLS Hosts"),
								Default:     util.Ptr("test"),
								Type:        util.Ptr("hostname"),
								Tags: &[]string{
									"tls",
								},
							},
							{
								DisplayName: util.Ptr("Cluster Issuer"),
								Default:     util.Ptr("issuer"),
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
				logger:           logger,
				controllerClient: fakeClient,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = []gin.Param{
				{Key: "appID", Value: tc.appID},
			}

			h.GetApp(c)

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
