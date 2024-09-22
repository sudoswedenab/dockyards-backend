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

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOrgs(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		sub      string
		expected []types.Organization
	}{
		{
			name: "test single",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "03582042-318e-4c1e-9728-755c5eaf4267",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "test",
										UID:  "89a3e0aa-7744-49af-ae7e-1461004c1598",
									},
								},
							},
						},
					},
				},
			},
			sub: "89a3e0aa-7744-49af-ae7e-1461004c1598",
			expected: []types.Organization{
				{
					ID:   "03582042-318e-4c1e-9728-755c5eaf4267",
					Name: "test",
				},
			},
		},
		{
			name: "test multiple organizations",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test1",
								UID:  "58c282c0-6a68-4ec8-9032-83d33f259bbe",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "user1",
										UID:  "2ca9e8a0-7b43-455d-867e-ed8bec4addfb",
									},
									{
										Name: "user2",
										UID:  "5cf0ed84-82f4-43fe-a3fb-b91f2ec7f0b1",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test2",
								UID:  "d327da4c-f8fe-4f85-93a1-258b729a40d2",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "user2",
										UID:  "5cf0ed84-82f4-43fe-a3fb-b91f2ec7f0b1",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test3",
								UID:  "5c13be53-fecd-467d-9546-d8ba3bb68103",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "user1",
										UID:  "2ca9e8a0-7b43-455d-867e-ed8bec4addfb",
									},
								},
							},
						},
					},
				},
			},
			sub: "2ca9e8a0-7b43-455d-867e-ed8bec4addfb",
			expected: []types.Organization{
				{
					ID:   "58c282c0-6a68-4ec8-9032-83d33f259bbe",
					Name: "test1",
				},
				{
					ID:   "5c13be53-fecd-467d-9546-d8ba3bb68103",
					Name: "test3",
				},
			},
		},
		{
			name: "test subject without organizations",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test1",
								UID:  "57236ef2-304c-4fa7-9aa7-e8019dfa3070",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "user1",
										UID:  "92770876-ae7f-493f-b3f8-7d9f0a45b656",
									},
									{
										Name: "user2",
										UID:  "29625697-69c7-4142-92dc-dccccfb5b824",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test2",
								UID:  "d327da4c-f8fe-4f85-93a1-258b729a40d2",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "user3",
										UID:  "df8ab98f-7866-4f4d-a9a6-7426879b7032",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test3",
								UID:  "5c13be53-fecd-467d-9546-d8ba3bb68103",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										Name: "user4",
										UID:  "d734d20f-e03e-44a8-89a5-8bd7f5d176d3",
									},
								},
							},
						},
					},
				},
			},
			sub: "59862b3c-61de-4362-aeac-36366035a914",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Organization{}, index.MemberReferencesField, index.ByMemberReferences).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/orgs"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetOrgs(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual []types.Organization
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body to json: %s", err)
			}

			if !cmp.Equal(tc.expected, actual) {
				t.Errorf("diff: %s", cmp.Diff(actual, tc.expected))
			}
		})
	}
}
