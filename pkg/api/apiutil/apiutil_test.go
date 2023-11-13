package apiutil_test

import (
	"context"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOwnerOrganization(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		object   client.Object
		expected v1alpha2.Organization
	}{
		{
			name: "test cluster with owner",
			lists: []client.ObjectList{
				&v1alpha2.OrganizationList{
					Items: []v1alpha2.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "42e4bdaf-d34b-44c4-bc7e-8a0d342153c9",
							},
							Status: v1alpha2.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
			},
			object: &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "e3094c41-9ab0-435f-b835-d41bb6df26bc",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: v1alpha2.GroupVersion.String(),
							Kind:       v1alpha2.OrganizationKind,
							Name:       "test",
							UID:        "42e4bdaf-d34b-44c4-bc7e-8a0d342153c9",
						},
					},
				},
			},
			expected: v1alpha2.Organization{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha2.GroupVersion.String(),
					Kind:       v1alpha2.OrganizationKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					UID:             "42e4bdaf-d34b-44c4-bc7e-8a0d342153c9",
					ResourceVersion: "999",
				},
				Status: v1alpha2.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
		},
		{
			name: "test cluster with multiple owners",
			lists: []client.ObjectList{
				&v1alpha2.OrganizationList{
					Items: []v1alpha2.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "eb92bcdb-ffae-4a8d-8581-94a664c60ea4",
							},
							Status: v1alpha2.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "baca1d93-8b3e-468d-becb-03f10166f83c",
							},
						},
					},
				},
				&v1alpha1.DeploymentList{
					Items: []v1alpha1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "7c751769-a739-42fa-aec1-ee6877f7a8b6",
							},
						},
					},
				},
			},
			object: &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "e3094c41-9ab0-435f-b835-d41bb6df26bc",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: v1alpha1.GroupVersion.String(),
							Kind:       v1alpha1.UserKind,
							Name:       "test",
							UID:        "baca1d93-8b3e-468d-becb-03f10166f83c",
						},
						{
							APIVersion: v1alpha2.GroupVersion.String(),
							Kind:       v1alpha2.OrganizationKind,
							Name:       "test",
							UID:        "eb92bcdb-ffae-4a8d-8581-94a664c60ea4",
						},
						{
							APIVersion: v1alpha1.GroupVersion.String(),
							Kind:       v1alpha1.DeploymentKind,
							Name:       "test",
							UID:        "7c751769-a739-42fa-aec1-ee6877f7a8b6",
						},
					},
				},
			},
			expected: v1alpha2.Organization{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha2.GroupVersion.String(),
					Kind:       v1alpha2.OrganizationKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					UID:             "eb92bcdb-ffae-4a8d-8581-94a664c60ea4",
					ResourceVersion: "999",
				},
				Status: v1alpha2.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			v1alpha2.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			actual, err := apiutil.GetOwnerOrganization(ctx, fakeClient, tc.object)
			if err != nil {
				t.Fatalf("error getting owner organization: %s", err)
			}

			if !cmp.Equal(actual, &tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(&tc.expected, actual))
			}
		})
	}
}
