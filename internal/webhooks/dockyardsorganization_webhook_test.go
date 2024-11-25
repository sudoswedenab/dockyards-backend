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

	"bitbucket.org/sudosweden/dockyards-backend/internal/webhooks"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
					MemberRefs: []dockyardsv1.OrganizationMemberReference{
						{
							Role: dockyardsv1.OrganizationMemberRoleSuperUser,
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
					MemberRefs: []dockyardsv1.OrganizationMemberReference{
						{
							Role: dockyardsv1.OrganizationMemberRoleSuperUser,
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
					MemberRefs: []dockyardsv1.OrganizationMemberReference{
						{
							Role: dockyardsv1.OrganizationMemberRoleSuperUser,
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
					MemberRefs: []dockyardsv1.OrganizationMemberReference{
						{
							Role: dockyardsv1.OrganizationMemberRoleSuperUser,
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
					MemberRefs: []dockyardsv1.OrganizationMemberReference{
						{
							Role: dockyardsv1.OrganizationMemberRoleUser,
						},
						{
							Role: dockyardsv1.OrganizationMemberRoleReader,
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
			scheme := runtime.NewScheme()

			_ = dockyardsv1.AddToScheme(scheme)

			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithLists(&tc.features).
				Build()

			webhook := webhooks.DockyardsOrganization{
				Client: c,
			}

			_, actual := webhook.ValidateCreate(context.Background(), &tc.dockyardsOrganization)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
