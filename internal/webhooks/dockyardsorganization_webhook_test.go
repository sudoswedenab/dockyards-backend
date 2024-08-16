package webhooks_test

import (
	"context"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/feature"
	"bitbucket.org/sudosweden/dockyards-backend/internal/webhooks"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestDockyardsOrganizationValidateCreate(t *testing.T) {
	tt := []struct {
		name                  string
		dockyardsOrganization dockyardsv1.Organization
		features              dockyardsv1.FeatureList
		expected              error
	}{
		{
			name: "test organization without auto-assign feature",
			dockyardsOrganization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "auto-assign",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
						},
					},
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.OrganizationKind).GroupKind(),
				"auto-assign",
				field.ErrorList{
					field.Invalid(field.NewPath("spec", "skipAutoAssign"), false, "feature is not enabled"),
				},
			),
		},
		{
			name: "test organization with auto-assign feature",
			dockyardsOrganization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "auto-assign",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
						},
					},
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      featurenames.FeatureOrganizationAutoAssign,
							Namespace: "testing",
						},
					},
				},
			},
		},
		{
			name: "test skip auto assign organization without auto-assign feature",
			dockyardsOrganization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "skip-auto-assign",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
						},
					},
					SkipAutoAssign: true,
				},
			},
		},
		{
			name: "test skip auto assign organization with auto-assign feature",
			dockyardsOrganization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "skip-auto-assign",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
						},
					},
					SkipAutoAssign: true,
				},
			},
			features: dockyardsv1.FeatureList{
				Items: []dockyardsv1.Feature{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      featurenames.FeatureOrganizationAutoAssign,
							Namespace: "testing",
						},
					},
				},
			},
		},
		{
			name: "test empty member references",
			dockyardsOrganization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-member-references",
				},
				Spec: dockyardsv1.OrganizationSpec{
					SkipAutoAssign: true,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.OrganizationKind).GroupKind(),
				"empty-member-references",
				field.ErrorList{
					field.Required(field.NewPath("spec", "memberRefs"), "must have at least one super user"),
				},
			),
		},
		{
			name: "test missing super user",
			dockyardsOrganization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-super-user",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleUser,
						},
						{
							Role: dockyardsv1.MemberRoleReader,
						},
					},
					SkipAutoAssign: true,
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.OrganizationKind).GroupKind(),
				"missing-super-user",
				field.ErrorList{
					field.Required(field.NewPath("spec", "memberRefs"), "must have at least one super user"),
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

			webhook := webhooks.DockyardsOrganization{}

			_, actual := webhook.ValidateCreate(context.Background(), &tc.dockyardsOrganization)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
