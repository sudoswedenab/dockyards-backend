package webhooks_test

import (
	"context"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/webhooks"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/google/go-cmp/cmp"
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
