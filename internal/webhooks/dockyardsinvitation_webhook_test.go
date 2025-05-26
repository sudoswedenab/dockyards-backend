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
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/webhooks"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func TestDockyardsInvitationWebhook_Create(t *testing.T) {
	ctx := t.Context()

	env := envtest.Environment{
		CRDDirectoryPaths: []string{
			path.Join("../../config/crd"),
		},
	}

	cfg, err := env.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := env.Stop()
		if err != nil {
			panic(err)
		}
	})

	scheme := runtime.NewScheme()

	_ = corev1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	existingUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "user@dockyards.dev",
		},
	}

	err = c.Create(ctx, &existingUser)
	if err != nil {
		t.Fatal(err)
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}

	err = c.Create(ctx, &namespace)
	if err != nil {
		t.Fatal(err)
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.OrganizationSpec{
			MemberRefs: []dockyardsv1.OrganizationMemberReference{
				{
					TypedLocalObjectReference: corev1.TypedLocalObjectReference{
						APIGroup: &dockyardsv1.GroupVersion.Group,
						Kind:     dockyardsv1.UserKind,
						Name:     existingUser.Name,
					},
					Role: dockyardsv1.OrganizationMemberRoleSuperUser,
					UID:  existingUser.UID,
				},
			},
			NamespaceRef: &corev1.LocalObjectReference{
				Name: namespace.Name,
			},
		},
	}

	err = c.Create(ctx, &organization)
	if err != nil {
		t.Fatal(err)
	}

	opts := manager.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
	}

	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		t.Fatal(err)
	}

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	webhook := webhooks.DockyardsInvitation{
		Client: mgr.GetClient(),
	}

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("could not wait for sync cache")
	}

	t.Run("test valid", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: namespace.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "test@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		}

		_, err := webhook.ValidateCreate(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test existing invitation", func(t *testing.T) {
		existing := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "existing@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleReader,
			},
		}

		err := c.Create(ctx, &existing)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &existing)
		if err != nil {
			t.Fatal(err)
		}

		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: namespace.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "existing@dockyards.dev",
				Role:  dockyardsv1.OrganizationMemberRoleUser,
			},
		}

		_, actual := webhook.ValidateCreate(ctx, &invitation)

		expected := apierrors.NewInvalid(
			dockyardsv1.GroupVersion.WithKind(dockyardsv1.InvitationKind).GroupKind(),
			invitation.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "email"),
					invitation.Spec.Email,
					"address already invited",
				),
			},
		)

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test existing user", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: namespace.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: existingUser.Spec.Email,
				Role:  dockyardsv1.OrganizationMemberRoleReader,
			},
		}

		_, actual := webhook.ValidateCreate(ctx, &invitation)

		expected := apierrors.NewInvalid(
			dockyardsv1.GroupVersion.WithKind(dockyardsv1.InvitationKind).GroupKind(),
			invitation.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "email"),
					invitation.Spec.Email,
					"user already member",
				),
			},
		)

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test invalid email", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: namespace.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Email: "test@",
				Role:  dockyardsv1.OrganizationMemberRoleReader,
			},
		}

		_, actual := webhook.ValidateCreate(ctx, &invitation)

		expected := apierrors.NewInvalid(
			dockyardsv1.GroupVersion.WithKind(dockyardsv1.InvitationKind).GroupKind(),
			invitation.Name,
			field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "email"),
					invitation.Spec.Email,
					"unable to parse as address",
				),
			},
		)

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
