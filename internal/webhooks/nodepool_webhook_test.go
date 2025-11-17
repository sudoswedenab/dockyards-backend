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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/webhooks"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDockyardsNodePoolValidateCreate(t *testing.T) {
	namespace := "testing"
	labels := map[string]string{
		dockyardsv1.LabelOrganizationName: "o",
		dockyardsv1.LabelClusterName:      "c",
	}

	tt := []struct {
		name              string
		dockyardsNodePool dockyardsv1.NodePool
		features          dockyardsv1.FeatureList
		expected          error
	}{
		{
			name: "test missing labels",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "label-validation",
					Namespace: namespace,
					Labels:    nil,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"label-validation",
				field.ErrorList{
					field.Invalid(field.NewPath("metadata", "labels"), map[string]string(nil), fmt.Sprintf("missing value for label '%s'", dockyardsv1.LabelOrganizationName)),
					field.Invalid(field.NewPath("metadata", "labels"), map[string]string(nil), fmt.Sprintf("missing value for label '%s'", dockyardsv1.LabelClusterName)),
				},
			),
		},
		{
			name: "test storage role disabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-role",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Storage: true,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"storage-role",
				field.ErrorList{
					field.Invalid(field.NewPath("spec", "storage"), true, "feature is not enabled"),
				},
			),
		},
		{
			name: "test storage role enabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-role",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Storage: true,
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureStorageRole),
							Namespace: namespace,
						},
					},
				},
			},
		},
		{
			name: "test storage resources disabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-resources",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("123"),
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"storage-resources",
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "storageResources"),
						[]dockyardsv1.NodePoolStorageResource{
							{
								Name:     "test",
								Quantity: resource.MustParse("123"),
							},
						},
						"feature is not enabled",
					),
				},
			),
		},
		{
			name: "test storage resources enabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-resources",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("123"),
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureStorageRole),
							Namespace: namespace,
						},
					},
				},
			},
		},
		{
			name: "test load-balancer role disabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "load-balancer-role",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					LoadBalancer: true,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"load-balancer-role",
				field.ErrorList{
					field.Invalid(field.NewPath("spec", "loadBalancer"), true, "feature is not enabled"),
				},
			),
		},
		{
			name: "test load-balancer role enabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "load-balancer-role",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					LoadBalancer: true,
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureLoadBalancerRole),
							Namespace: namespace,
						},
					},
				},
			},
		},
		{
			name: "test invalid storage resource name",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-storage-resource-name",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "Test Invalid Name",
							Quantity: resource.MustParse("123"),
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureStorageRole),
							Namespace: namespace,
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"invalid-storage-resource-name",
				field.ErrorList{
					field.Invalid(field.NewPath("spec", "storageResources[0]", "name"), "Test Invalid Name", "not a valid name"),
				},
			),
		},
		{
			name: "test duplicated storage resource name",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicated-storage-resource-name",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("123"),
						},
						{
							Name:     "test",
							Quantity: resource.MustParse("234"),
						},
						{
							Name:     "test",
							Quantity: resource.MustParse("345"),
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureStorageRole),
							Namespace: namespace,
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"duplicated-storage-resource-name",
				field.ErrorList{
					field.Duplicate(field.NewPath("spec", "storageResources").Child("name"), "test"),
				},
			),
		},
		{
			name: "test storage resource type host path disabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-storage-resource-type-host-path",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("123"),
							Type:     dockyardsv1.StorageResourceTypeHostPath,
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureStorageRole),
							Namespace: namespace,
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"test-storage-resource-type-host-path",
				field.ErrorList{
					field.Invalid(field.NewPath("spec", "storageResources").Index(0).Child("type"), dockyardsv1.StorageResourceTypeHostPath, "feature is not enabled"),
				},
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()

			_ = dockyardsv1.AddToScheme(scheme)

			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithLists(&tc.features).
				Build()

			webhook := webhooks.DockyardsNodePool{
				Client: c,
			}

			_, actual := webhook.ValidateCreate(context.Background(), &tc.dockyardsNodePool)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestDockyardsNodePoolValidateUpdate(t *testing.T) {
	namespace := "testing"
	labels := map[string]string{
		dockyardsv1.LabelOrganizationName: "o",
		dockyardsv1.LabelClusterName:      "c",
	}

	tt := []struct {
		name        string
		oldNodePool dockyardsv1.NodePool
		newNodePool dockyardsv1.NodePool
		features    dockyardsv1.FeatureList
		expected    error
	}{
		{
			name: "test resources update",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resources-update",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("1"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceStorage: resource.MustParse("3G"),
					},
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resources-update",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("2"),
						corev1.ResourceMemory:  resource.MustParse("3Gi"),
						corev1.ResourceStorage: resource.MustParse("4G"),
					},
				},
			},
		},
		{
			name: "test remove required label",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-label-removal",
					Namespace: namespace,
					Labels:    labels,
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-label-removal",
					Namespace: namespace,
					Labels:    map[string]string{dockyardsv1.LabelClusterName: "c"},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"test-label-removal",
				field.ErrorList{
					field.Invalid(field.NewPath("metadata", "labels"), map[string]string{dockyardsv1.LabelClusterName: "c"}, fmt.Sprintf("missing value for label '%s'", dockyardsv1.LabelOrganizationName)),
				},
			),
		},
		{
			name: "test immutable-resources feature",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-immutable-resources-feature",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("1"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceStorage: resource.MustParse("3G"),
					},
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-immutable-resources-feature",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Resources: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("2"),
						corev1.ResourceMemory:  resource.MustParse("3Gi"),
						corev1.ResourceStorage: resource.MustParse("4G"),
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureImmutableResources),
							Namespace: namespace,
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"test-immutable-resources-feature",
				field.ErrorList{
					field.Forbidden(field.NewPath("spec", "resources"), "immutable-resources feature is enabled"),
				},
			),
		},
		{
			name: "test scaling down control plane to zero replicas",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-control-plane-zero-replicas",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					ControlPlane: true,
					Replicas:     ptr.To(int32(1)),
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-control-plane-zero-replicas",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					ControlPlane: true,
					Replicas:     ptr.To(int32(0)),
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"test-control-plane-zero-replicas",
				field.ErrorList{
					field.Invalid(field.NewPath("spec", "replicas"), 0, "must be at least 1 for control plane"),
				},
			),
		},
		{
			name: "test scaling down worker to zero replicas",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-worker-zero-replicas",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Replicas: ptr.To(int32(2)),
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-worker-zero-replicas",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					Replicas: ptr.To(int32(0)),
				},
			},
		},
		{
			name: "test storage resource update",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-storage-resource-update",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("12Gi"),
						},
					},
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-storage-resource-update",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("123Gi"),
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
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
		{
			name: "test immutable storage resources enabled",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-immutable-resources-enabled",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("12Gi"),
						},
					},
				},
			},
			newNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-immutable-resources-enabled",
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: dockyardsv1.NodePoolSpec{
					StorageResources: []dockyardsv1.NodePoolStorageResource{
						{
							Name:     "test",
							Quantity: resource.MustParse("123Gi"),
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureImmutableStorageResources),
							Namespace: namespace,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      string(featurenames.FeatureStorageRole),
							Namespace: namespace,
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(),
				"test-immutable-resources-enabled",
				field.ErrorList{
					field.Forbidden(field.NewPath("spec", "storageResources"), "immutable-storage-resources feature is enabled"),
				},
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()

			_ = dockyardsv1.AddToScheme(scheme)

			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithLists(&tc.features).
				Build()

			webhook := webhooks.DockyardsNodePool{
				Client: c,
			}

			_, actual := webhook.ValidateUpdate(context.Background(), &tc.oldNodePool, &tc.newNodePool)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
