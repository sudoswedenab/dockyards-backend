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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/webhooks"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDockyardsMemberValidateCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	webhook := webhooks.DockyardsMember{}

	tt := []struct {
		name       string
		makeMember func() dockyardsv1.Member
		expected   func(dockyardsv1.Member) error
	}{
		{
			name: "valid member",
			makeMember: func() dockyardsv1.Member {
				labels := validMemberLabels("valid-user")

				return newTestMember("valid-member", "valid-user", labels)
			},
			expected: func(dockyardsv1.Member) error {
				return nil
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			member := tc.makeMember()
			expected := tc.expected(member)

			_, actual := webhook.ValidateCreate(ctx, member.DeepCopy())
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}

func TestDockyardsMemberValidateUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	webhook := webhooks.DockyardsMember{}

	tt := []struct {
		name  string
		setup func() (runtime.Object, runtime.Object, error)
	}{
		{
			name: "valid member update",
			setup: func() (runtime.Object, runtime.Object, error) {
				labels := validMemberLabels("valid-update")
				oldMember := newTestMember("valid-update", "valid-update", labels)

				newMember := oldMember.DeepCopy()
				newMember.Annotations = map[string]string{"updated": "true"}

				return oldMember.DeepCopy(), newMember, nil
			},
		},
		{
			name: "immutable user reference",
			setup: func() (runtime.Object, runtime.Object, error) {
				labels := validMemberLabels("immutable-old")
				oldMember := newTestMember("immutable-user", "immutable-old", labels)

				newLabels := validMemberLabels("immutable-new")
				newMember := newTestMember("immutable-user", "immutable-new", newLabels)

				newMemberObj := newMember.DeepCopy()

				return oldMember.DeepCopy(), newMemberObj, apierrors.NewInvalid(
					dockyardsv1.GroupVersion.WithKind(dockyardsv1.MemberKind).GroupKind(),
					newMemberObj.Name,
					field.ErrorList{
						field.Invalid(
							field.NewPath("spec", "userRef"),
							newMemberObj.Spec.UserRef,
							"field is immutable",
						),
					},
				)
			},
		},
		{
			name: "unexpected new object type",
			setup: func() (runtime.Object, runtime.Object, error) {
				labels := validMemberLabels("type-check")
				oldMember := newTestMember("type-check", "type-check", labels)

				return oldMember.DeepCopy(), &dockyardsv1.Organization{}, apierrors.NewBadRequest("new object has an unexpected type")
			},
		},
		{
			name: "unexpected old object type",
			setup: func() (runtime.Object, runtime.Object, error) {
				labels := validMemberLabels("type-check")
				newMember := newTestMember("type-check", "type-check", labels)

				return &dockyardsv1.Organization{}, newMember.DeepCopy(), apierrors.NewInternalError(errors.New("existing object has an unexpected type"))
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			oldObj, newObj, expected := tc.setup()

			_, actual := webhook.ValidateUpdate(ctx, oldObj, newObj)
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}

func validMemberLabels(userName string) map[string]string {
	return map[string]string{
		dockyardsv1.LabelOrganizationName: "org",
		dockyardsv1.LabelUserName:         userName,
		dockyardsv1.LabelRoleName:         string(dockyardsv1.RoleUser),
	}
}

func newTestMember(objectName, userName string, labels map[string]string) dockyardsv1.Member {
	group := dockyardsv1.GroupVersion.Group

	return dockyardsv1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name:   objectName,
			Labels: labels,
		},
		Spec: dockyardsv1.MemberSpec{
			Role: dockyardsv1.RoleUser,
			UserRef: corev1.TypedLocalObjectReference{
				APIGroup: &group,
				Kind:     dockyardsv1.UserKind,
				Name:     userName,
			},
		},
	}
}

func TestDockyardsMemberDefault(t *testing.T) {
	ctx := t.Context()
	scheme := runtime.NewScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	t.Run("test labels", func(t *testing.T) {
		organizationList := dockyardsv1.OrganizationList{
			Items: []dockyardsv1.Organization{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: dockyardsv1.OrganizationSpec{
						NamespaceRef: &corev1.LocalObjectReference{
							Name: "testing",
						},
					},
				},
			},
		}

		c := fake.NewClientBuilder().
			WithScheme(scheme).
			WithLists(&organizationList).
			Build()

		member := dockyardsv1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "testing",
			},
			Spec: dockyardsv1.MemberSpec{
				Role: dockyardsv1.RoleSuperUser,
				UserRef: corev1.TypedLocalObjectReference{
					Name: "super-user",
				},
			},
		}

		webhook := webhooks.DockyardsMember{
			Client: c,
		}

		err := webhook.Default(ctx, &member)
		if err != nil {
			t.Fatal(err)
		}

		expected := map[string]string{
			dockyardsv1.LabelOrganizationName: "test",
			dockyardsv1.LabelRoleName:         "SuperUser",
			dockyardsv1.LabelUserName:         "super-user",
		}

		if !cmp.Equal(member.Labels, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, member.Labels))
		}
	})
}
