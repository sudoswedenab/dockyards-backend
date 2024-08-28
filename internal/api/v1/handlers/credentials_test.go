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
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOrgCredentials(t *testing.T) {
	tt := []struct {
		name           string
		sub            string
		organizationID string
		lists          []client.ObjectList
		expected       []v1.Credential
	}{
		{
			name:           "test single credential",
			sub:            "654202f2-44f6-4fa6-873b-0b9817d3957c",
			organizationID: "af2224ee-fd4b-4e6c-8ff6-21c2d1ddcc5c",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "af2224ee-fd4b-4e6c-8ff6-21c2d1ddcc5c",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										UID: "654202f2-44f6-4fa6-873b-0b9817d3957c",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
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
								UID:       "54376668-876c-43d7-8d29-2ef37ccab831",
							},
							Type: DockyardsSecretTypeCredential,
						},
					},
				},
			},
			expected: []v1.Credential{
				{
					ID:           "54376668-876c-43d7-8d29-2ef37ccab831",
					Name:         "test",
					Organization: "test",
				},
			},
		},
		{
			name:           "test several secret types",
			sub:            "41ae3267-da66-4be0-b2ac-57a60549ff57",
			organizationID: "8afac404-d43a-4253-a102-a90ff80fa13c",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "8afac404-d43a-4253-a102-a90ff80fa13c",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										UID: "41ae3267-da66-4be0-b2ac-57a60549ff57",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&corev1.SecretList{
					Items: []corev1.Secret{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "dockyards-io-credential",
								Namespace: "testing",
								UID:       "3cca83a8-7848-40ad-aa89-916a28f6016d",
							},
							Type: DockyardsSecretTypeCredential,
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "kubernetes-io-ssh-auth",
								Namespace: "testing",
								UID:       "bf8fc71c-3278-40fe-a452-ed0b1ee189b8",
							},
							Type: corev1.SecretTypeSSHAuth,
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "kubernetes-io-tls",
								Namespace: "testing",
								UID:       "224a442e-515f-4042-9e03-10de6b827ecf",
							},
							Type: corev1.SecretTypeTLS,
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "opaque",
								Namespace: "testing",
								UID:       "1efe00b7-9e6a-425d-88ea-99cc41eb6011",
							},
							Type: corev1.SecretTypeOpaque,
						},
					},
				},
			},
			expected: []v1.Credential{
				{
					ID:           "3cca83a8-7848-40ad-aa89-916a28f6016d",
					Name:         "dockyards-io-credential",
					Organization: "test",
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
				WithIndex(&dockyardsv1.Organization{}, index.UIDField, index.ByUID).
				WithIndex(&corev1.Secret{}, index.SecretTypeField, index.BySecretType).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationID, "credentials"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("organizationID", tc.organizationID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetOrgCredentials(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual []v1.Credential
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
