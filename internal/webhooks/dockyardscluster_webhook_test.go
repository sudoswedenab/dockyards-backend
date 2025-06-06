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

package webhooks_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/webhooks"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestDockyardsClusterValidateCreate(t *testing.T) {
	tt := []struct {
		name             string
		dockyardsCluster dockyardsv1.Cluster
		expected         error
	}{
		{
			name: "test cluster with organization owner",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "with-organization-owner",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "8004bcb8-146c-445d-a95d-0ab7842184d8",
						},
					},
				},
			},
		},
		{
			name: "test cluster without organization owner",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-organization-owner",
					Namespace: "testing",
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(),
				"no-organization-owner",
				field.ErrorList{
					field.Required(
						field.NewPath("metadata", "ownerReferences"),
						"must have organization owner reference",
					),
				},
			),
		},
		{
			name: "test with internal ip allocation",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-with-internal-ip-allocation",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "a80777fc-078b-47dd-9252-0802990aedf8",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					AllocateInternalIP: true,
				},
			},
		},
		{
			name: "test custom pod subnets",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "custom-pod-subnets",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "cbfab09e-d289-4ad9-b05f-ec465374ca2b",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					PodSubnets: []string{
						"192.168.0.0/16",
						"fc00:192:168::/56",
					},
				},
			},
		},
		{
			name: "test invalid pod subnets",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-pod-subnets",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "cbfab09e-d289-4ad9-b05f-ec465374ca2b",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					PodSubnets: []string{
						"192.168.0.0",
						"fc00:192:168::1",
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(),
				"invalid-pod-subnets",
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "podSubnets").Index(0),
						"192.168.0.0",
						"unable to parse pod subnet as prefix",
					),
					field.Invalid(
						field.NewPath("spec", "podSubnets").Index(1),
						"fc00:192:168::1",
						"unable to parse pod subnet as prefix",
					),
				},
			),
		},
		{
			name: "test custom service subnets",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "custom-service-subnets",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "f8bf51bb-6e67-404d-acb7-350563df68f1",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					ServiceSubnets: []string{
						"10.96.0.0/12",
						"fc00:10:96::/112",
					},
				},
			},
		},
		{
			name: "test invalid service subnets",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalid-service-subnets",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "8358eee3-7b2e-4d48-9bba-a5d6fb3ac719",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					ServiceSubnets: []string{
						"10.96.0.0",
						"fc00:10:96::1",
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(),
				"invalid-service-subnets",
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "serviceSubnets").Index(0),
						"10.96.0.0",
						"unable to parse service subnet as prefix",
					),
					field.Invalid(
						field.NewPath("spec", "serviceSubnets").Index(1),
						"fc00:10:96::1",
						"unable to parse service subnet as prefix",
					),
				},
			),
		},
		{
			name: "test overlapping subnets",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "overlapping-subnets",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "testing",
							UID:        "8358eee3-7b2e-4d48-9bba-a5d6fb3ac719",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					PodSubnets:  []string{
						"10.100.0.0/16",
					},
					ServiceSubnets:  []string{
						"10.96.0.0/12",
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(),
				"overlapping-subnets",
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "serviceSubnets").Index(0),
						"10.96.0.0/12",
						"subnet overlaps with prefix 10.100.0.0/16",
					),
				},
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhook := webhooks.DockyardsCluster{}

			_, actual := webhook.ValidateCreate(context.Background(), &tc.dockyardsCluster)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestDockyardsClusterValidateDelete(t *testing.T) {
	tt := []struct {
		name             string
		dockyardsCluster dockyardsv1.Cluster
		expected         error
	}{
		{
			name: "test empty cluster",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty",
					Namespace: "testing",
				},
			},
		},
		{
			name: "test cluster with block deletion",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "block-deletion",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					BlockDeletion: true,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(),
				"block-deletion",
				field.ErrorList{
					field.Forbidden(
						field.NewPath("spec", "blockDeletion"),
						"deletion is blocked",
					),
				},
			),
		},
		{
			name: "test expired cluster with block deletion",
			dockyardsCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{
						Time: time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC),
					},
					Name:      "test-expired-block-deletion",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					BlockDeletion: true,
					Duration: &metav1.Duration{
						Duration: time.Minute * 15,
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhook := webhooks.DockyardsCluster{}

			_, actual := webhook.ValidateDelete(context.Background(), &tc.dockyardsCluster)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestDockyardsClusterValidateUpdate(t *testing.T) {
	tt := []struct {
		name       string
		oldCluster dockyardsv1.Cluster
		newCluster dockyardsv1.Cluster
		expected   error
	}{
		{
			name: "test enable internal ip allocation",
			oldCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "enable-internal-ip-allocation",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "4ec0c0ac-4bd1-44da-b514-4eefa9e7fba5",
						},
					},
				},
			},
			newCluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "enable-internal-ip-allocation",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "4ec0c0ac-4bd1-44da-b514-4eefa9e7fba5",
						},
					},
				},
				Spec: dockyardsv1.ClusterSpec{
					AllocateInternalIP: true,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(),
				"enable-internal-ip-allocation",
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "allocateInternalIP"),
						true,
						"field is immutable",
					),
				},
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhook := webhooks.DockyardsCluster{}

			_, actual := webhook.ValidateUpdate(context.Background(), &tc.oldCluster, &tc.newCluster)
			if !cmp.Equal(actual, tc.expected) {
				t.Fatalf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
