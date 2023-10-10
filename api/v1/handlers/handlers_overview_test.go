package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOverview(t *testing.T) {
	tt := []struct {
		name               string
		sub                string
		lists              []client.ObjectList
		clustermockOptions []clustermock.MockOption
		expected           v1.Overview
	}{
		{
			name: "test missing membership",
			sub:  "1f85ab1d-382b-4cd8-a860-c3108b6eefd2",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "6b60a111-45a9-4f20-b3f4-c431f47daa7b",
							},
						},
					},
				},
			},
			expected: v1.Overview{},
		},
		{
			name: "test single organization",
			sub:  "23aa679e-e3c8-47f8-bad6-2284b92b83e7",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "01cdc762-a115-43c6-9770-6587087d6fb8",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "test",
										UID:  "23aa679e-e3c8-47f8-bad6-2284b92b83e7",
									},
								},
							},
						},
					},
				},
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "23aa679e-e3c8-47f8-bad6-2284b92b83e7",
							},
							Spec: v1alpha1.UserSpec{
								Email: "test@dockyards.dev",
							},
						},
					},
				},
			},
			expected: v1.Overview{
				Organizations: []v1.OrganizationOverview{
					{
						Id:   "01cdc762-a115-43c6-9770-6587087d6fb8",
						Name: "test",
						Users: &[]v1.UserOverview{
							{
								Id:    "23aa679e-e3c8-47f8-bad6-2284b92b83e7",
								Email: "test@dockyards.dev",
							},
						},
					},
				},
			},
		},
		{
			name: "test multiple organization",
			sub:  "e27df599-93cd-4e30-93f1-98399e3e7237",
			lists: []client.ObjectList{
				&v1alpha1.OrganizationList{
					Items: []v1alpha1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test1",
								Namespace: "testing",
								UID:       "8d2c74e5-efee-4b70-8f9d-e3535ba2d1f9",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "user1",
										UID:  "e27df599-93cd-4e30-93f1-98399e3e7237",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test2",
								Namespace: "testing",
								UID:       "522941b7-2883-4a9f-83a3-04f32b033334",
							},
							Spec: v1alpha1.OrganizationSpec{
								MemberRefs: []v1alpha1.UserReference{
									{
										Name: "user2",
										UID:  "e9f13deb-de5e-4737-94e6-b3e1a7dd7b1b",
									},
								},
							},
						},
					},
				},
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "user1",
								Namespace: "testing",
								UID:       "e27df599-93cd-4e30-93f1-98399e3e7237",
							},
							Spec: v1alpha1.UserSpec{
								Email: "user1@dockyards.dev",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "user2",
								Namespace: "testing",
								UID:       "e9f13deb-de5e-4737-94e6-b3e1a7dd7b1b",
							},
							Spec: v1alpha1.UserSpec{
								Email: "user2@dockyards.dev",
							},
						},
					},
				},
			},
			expected: v1.Overview{
				Organizations: []v1.OrganizationOverview{
					{
						Id:   "8d2c74e5-efee-4b70-8f9d-e3535ba2d1f9",
						Name: "test1",
						Users: &[]v1.UserOverview{
							{
								Id:    "e27df599-93cd-4e30-93f1-98399e3e7237",
								Email: "user1@dockyards.dev",
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
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			h := handler{
				clusterService:   clustermock.NewMockClusterService(tc.clustermockOptions...),
				logger:           logger,
				controllerClient: fakeClient,
				namespace:        "testing",
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			u := url.URL{
				Path: path.Join("/v1/refresh"),
			}

			c.Set("sub", tc.sub)
			c.Request = &http.Request{
				Method: http.MethodPost,
				URL:    &u,
			}

			h.GetOverview(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual v1.Overview
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
