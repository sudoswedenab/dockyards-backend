// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiutil_test

import (
	"context"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOwnerOrganization(t *testing.T) {
	tt := []struct {
		name     string
		lists    []client.ObjectList
		object   client.Object
		expected dockyardsv1.Organization
	}{
		{
			name: "test cluster with owner",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "42e4bdaf-d34b-44c4-bc7e-8a0d342153c9",
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
			},
			object: &dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "e3094c41-9ab0-435f-b835-d41bb6df26bc",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "42e4bdaf-d34b-44c4-bc7e-8a0d342153c9",
						},
					},
				},
			},
			expected: dockyardsv1.Organization{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.OrganizationKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					UID:             "42e4bdaf-d34b-44c4-bc7e-8a0d342153c9",
					ResourceVersion: "999",
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: &corev1.LocalObjectReference{
						Name: "testing",
					},
				},
			},
		},
		{
			name: "test cluster with multiple owners",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "eb92bcdb-ffae-4a8d-8581-94a664c60ea4",
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: &corev1.LocalObjectReference{
									Name: "testing",
								},
							},
						},
					},
				},
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "baca1d93-8b3e-468d-becb-03f10166f83c",
							},
						},
					},
				},
				&dockyardsv1.DeploymentList{
					Items: []dockyardsv1.Deployment{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "7c751769-a739-42fa-aec1-ee6877f7a8b6",
							},
						},
					},
				},
			},
			object: &dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "e3094c41-9ab0-435f-b835-d41bb6df26bc",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.UserKind,
							Name:       "test",
							UID:        "baca1d93-8b3e-468d-becb-03f10166f83c",
						},
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "eb92bcdb-ffae-4a8d-8581-94a664c60ea4",
						},
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.DeploymentKind,
							Name:       "test",
							UID:        "7c751769-a739-42fa-aec1-ee6877f7a8b6",
						},
					},
				},
			},
			expected: dockyardsv1.Organization{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.OrganizationKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					UID:             "eb92bcdb-ffae-4a8d-8581-94a664c60ea4",
					ResourceVersion: "999",
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: &corev1.LocalObjectReference{
						Name: "testing",
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			dockyardsv1.AddToScheme(scheme)
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
		expected dockyardsv1.Cluster
	}{
		{
			name: "test nodepool with owner",
			lists: []client.ObjectList{
				&dockyardsv1.ClusterList{
					Items: []dockyardsv1.Cluster{
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
			object: &dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "8a6d302a-89d2-418b-bf01-0b907cdd8d42",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.ClusterKind,
							Name:       "test",
							UID:        "2e83235e-478f-4c58-804f-1d153b6457b5",
						},
					},
				},
			},
			expected: dockyardsv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
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
			dockyardsv1.AddToScheme(scheme)
			dockyardsv1.AddToScheme(scheme)
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
		expected *dockyardsv1.NodePool
	}{
		{
			name: "test node",
			object: &dockyardsv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.NodePoolKind,
							Name:       "test",
							UID:        "ebafc27f-b2de-418c-8c50-2ca83bd7e492",
						},
					},
				},
			},
			lists: []client.ObjectList{
				&dockyardsv1.NodePoolList{
					Items: []dockyardsv1.NodePool{
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
			expected: &dockyardsv1.NodePool{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.NodePoolKind,
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
			dockyardsv1.AddToScheme(scheme)
			dockyardsv1.AddToScheme(scheme)
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
				&dockyardsv1.FeatureList{
					Items: []dockyardsv1.Feature{
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
				&dockyardsv1.FeatureList{
					Items: []dockyardsv1.Feature{
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
			dockyardsv1.AddToScheme(scheme)
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
		organizations dockyardsv1.OrganizationList
		namespace     string
		expected      *dockyardsv1.Organization
	}{
		{
			name: "test empty",
		},
		{
			name: "test organization with namespace",
			organizations: dockyardsv1.OrganizationList{
				Items: []dockyardsv1.Organization{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test",
						},
						Status: dockyardsv1.OrganizationStatus{
							NamespaceRef: &corev1.LocalObjectReference{
								Name: "testing",
							},
						},
					},
				},
			},
			namespace: "testing",
			expected: &dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					ResourceVersion: "999",
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: &corev1.LocalObjectReference{
						Name: "testing",
					},
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
			dockyardsv1.AddToScheme(scheme)
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

func TestSetWorkloadReference(t *testing.T) {
	tt := []struct {
		name          string
		references    []dockyardsv1.WorkloadReference
		reference     dockyardsv1.WorkloadReference
		expected      []dockyardsv1.WorkloadReference
		expectChanged bool
	}{
		{
			name: "test empty",
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind: "Test",
					Name: "test",
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "test existing",
			references: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
			},
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind: "Test",
					Name: "test",
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
			},
		},
		{
			name: "test new",
			references: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
			},
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind: "Test",
					Name: "new",
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "new",
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "test namespace",
			references: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
			},
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind:      "Test",
					Name:      "test",
					Namespace: ptr.To("testing"),
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind:      "Test",
						Name:      "test",
						Namespace: ptr.To("testing"),
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "test urls",
			references: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
				},
			},
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind: "Test",
					Name: "test",
				},
				URLs: []string{
					"http://localhost:8080",
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
					URLs: []string{
						"http://localhost:8080",
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "test existing urls",
			references: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
					URLs: []string{
						"http://localhost:1234",
					},
				},
			},
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind: "Test",
					Name: "test",
				},
				URLs: []string{
					"http://localhost:1234",
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
					URLs: []string{
						"http://localhost:1234",
					},
				},
			},
		},
		{
			name: "test changed urls",
			references: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
					URLs: []string{
						"http://localhost:8080",
					},
				},
			},
			reference: dockyardsv1.WorkloadReference{
				TypedObjectReference: corev1.TypedObjectReference{
					Kind: "Test",
					Name: "test",
				},
				URLs: []string{
					"https://localhost:6443",
				},
			},
			expected: []dockyardsv1.WorkloadReference{
				{
					TypedObjectReference: corev1.TypedObjectReference{
						Kind: "Test",
						Name: "test",
					},
					URLs: []string{
						"https://localhost:6443",
					},
				},
			},
			expectChanged: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := apiutil.SetWorkloadReference(&tc.references, tc.reference)
			if actual != tc.expectChanged {
				t.Errorf("expected change %t, got %t", tc.expectChanged, actual)
			}

			if !cmp.Equal(tc.references, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, tc.references))
			}
		})
	}
}
