package apiutil_test

import (
	"context"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

func TestGetOwnerCluster(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		object   client.Object
		expected v1alpha1.Cluster
	}{
		{
			name: "test nodepool with owner",
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "2e83235e-478f-4c58-804f-1d153b6457b5",
							},
						},
					},
				},
			},
			object: &v1alpha1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "8a6d302a-89d2-418b-bf01-0b907cdd8d42",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: v1alpha1.GroupVersion.String(),
							Kind:       v1alpha1.ClusterKind,
							Name:       "test",
							UID:        "2e83235e-478f-4c58-804f-1d153b6457b5",
						},
					},
				},
			},
			expected: v1alpha1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.ClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					Namespace:       "testing",
					UID:             "2e83235e-478f-4c58-804f-1d153b6457b5",
					ResourceVersion: "999",
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

			actual, err := apiutil.GetOwnerCluster(ctx, fakeClient, tc.object)
			if err != nil {
				t.Fatalf("error getting owner organization: %s", err)
			}

			if !cmp.Equal(actual, &tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(&tc.expected, actual))
			}
		})
	}
}

func TestGetOwnerNodePool(t *testing.T) {
	tt := []struct {
		name     string
		object   client.Object
		lists    []client.ObjectList
		expected *v1alpha1.NodePool
	}{
		{
			name: "test node",
			object: &v1alpha1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: v1alpha1.GroupVersion.String(),
							Kind:       v1alpha1.NodePoolKind,
							Name:       "test",
							UID:        "ebafc27f-b2de-418c-8c50-2ca83bd7e492",
						},
					},
				},
			},
			lists: []client.ObjectList{
				&v1alpha1.NodePoolList{
					Items: []v1alpha1.NodePool{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "ebafc27f-b2de-418c-8c50-2ca83bd7e492",
							},
						},
					},
				},
			},
			expected: &v1alpha1.NodePool{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.NodePoolKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					Namespace:       "testing",
					UID:             "ebafc27f-b2de-418c-8c50-2ca83bd7e492",
					ResourceVersion: "999",
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

			actual, err := apiutil.GetOwnerNodePool(ctx, fakeClient, tc.object)
			if err != nil {
				t.Fatalf("error getting owner node pool: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestIsFeatureEnabled(t *testing.T) {
	tt := []struct {
		name        string
		featureName featurenames.FeatureName
		lists       []client.ObjectList
		expected    bool
	}{
		{
			name:     "test empty",
			expected: false,
		},
		{
			name:        "test load balancer role",
			featureName: featurenames.FeatureLoadBalancerRole,
			lists: []client.ObjectList{
				&v1alpha1.FeatureList{
					Items: []v1alpha1.Feature{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      string(featurenames.FeatureLoadBalancerRole),
								Namespace: "testing",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name:        "test undefined role",
			featureName: featurenames.FeatureName("undefined-role"),
			lists: []client.ObjectList{
				&v1alpha1.FeatureList{
					Items: []v1alpha1.Feature{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      string(featurenames.FeatureLoadBalancerRole),
								Namespace: "testing",
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			scheme := scheme.Scheme
			v1alpha2.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			actual, err := apiutil.IsFeatureEnabled(ctx, fakeClient, tc.featureName, "testing")
			if err != nil {
				t.Fatalf("unexpected error testing feature: %s", err)
			}

			if actual != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, actual)
			}
		})
	}
}

func TestGetNamespaceOrganization(t *testing.T) {
	tt := []struct {
		name          string
		organizations v1alpha2.OrganizationList
		namespace     string
		expected      *v1alpha2.Organization
	}{
		{
			name: "test empty",
		},
		{
			name: "test organization with namespace",
			organizations: v1alpha2.OrganizationList{
				Items: []v1alpha2.Organization{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test",
						},
						Status: v1alpha2.OrganizationStatus{
							NamespaceRef: "testing",
						},
					},
				},
			},
			namespace: "testing",
			expected: &v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					ResourceVersion: "999",
				},
				Status: v1alpha2.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
		},
		{
			name:      "test namespace without organization",
			namespace: "testing",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			scheme := scheme.Scheme
			v1alpha2.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(&tc.organizations).
				Build()

			actual, err := apiutil.GetNamespaceOrganization(ctx, fakeClient, tc.namespace)
			if err != nil {
				t.Fatalf("error getting namespace organization: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestCanUserGet(t *testing.T) {
	tt := []struct {
		name           string
		user           v1alpha1.User
		namespacedName types.NamespacedName
		lists          []client.ObjectList
		expected       bool
	}{
		{
			name: "test dockyards cluster",
			user: v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{},
			},
			namespacedName: types.NamespacedName{
				Namespace: "testing",
				Name:      "test",
			},
			lists: []client.ObjectList{
				&v1alpha1.ClusterList{
					Items: []v1alpha1.Cluster{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme := scheme.Scheme
			_ = v1alpha1.AddToScheme(scheme)
			_ = v1alpha2.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			actual, _ := apiutil.CanUserGet(context.Background(), fakeClient, &tc.user, tc.namespacedName)
			if actual != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, actual)
			}
		})
	}
}
