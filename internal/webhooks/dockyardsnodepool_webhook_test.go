package webhooks_test

import (
	"context"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/feature"
	"bitbucket.org/sudosweden/dockyards-backend/internal/webhooks"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestDockyardsNodePoolValidateCreate(t *testing.T) {
	tt := []struct {
		name              string
		dockyardsNodePool dockyardsv1.NodePool
		features          dockyardsv1.FeatureList
		expected          error
	}{
		{
			name: "test storage role disabled",
			dockyardsNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-role",
					Namespace: "testing",
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
					Namespace: "testing",
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
							Namespace: "testing",
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
					Namespace: "testing",
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
					Namespace: "testing",
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
							Namespace: "testing",
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
					Namespace: "testing",
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
					Namespace: "testing",
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
							Namespace: "testing",
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
					Namespace: "testing",
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
							Namespace: "testing",
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
					Namespace: "testing",
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
							Namespace: "testing",
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			for _, item := range tc.features.Items {
				feature.Enable(featurenames.FeatureName(item.Name))

				defer feature.Disable(featurenames.FeatureName(item.Name))
			}

			webhook := webhooks.DockyardsNodePool{}

			_, actual := webhook.ValidateCreate(context.Background(), &tc.dockyardsNodePool)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestDockyardsNodePoolValidateUpdate(t *testing.T) {
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
					Namespace: "testing",
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
					Namespace: "testing",
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
			name: "test immutable-resources feature",
			oldNodePool: dockyardsv1.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-immutable-resources-feature",
					Namespace: "testing",
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
					Namespace: "testing",
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
							Namespace: "testing",
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			for _, item := range tc.features.Items {
				feature.Enable(featurenames.FeatureName(item.Name))

				defer feature.Disable(featurenames.FeatureName(item.Name))
			}

			webhook := webhooks.DockyardsNodePool{}

			_, actual := webhook.ValidateUpdate(context.Background(), &tc.oldNodePool, &tc.newNodePool)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
