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

	"github.com/google/go-cmp/cmp"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/webhooks"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestDockyardsUserValidateCreate(t *testing.T) {
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
			Email: "existing@dockyards.dev",
		},
	}

	err = c.Create(ctx, &existingUser)
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := manager.New(cfg, manager.Options{Scheme: scheme})
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

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("could not wait for cache sync")
	}

	tt := []struct {
		name           string
		allowedDomains []string
		dockyardsUser  dockyardsv1.User
		expected       error
	}{
		{
			name: "test without allowed domains",
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-without-allowed-domains",
				},
				Spec: dockyardsv1.UserSpec{
					Email: "test-without-allowed-domains@dockyards.dev",
				},
			},
		},
		{
			name: "test with allowed domain",
			allowedDomains: []string{
				"@dockyards.dev",
			},
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-with-allowed-domain",
				},
				Spec: dockyardsv1.UserSpec{
					Email: "test-with-allowed-domain@dockyards.dev",
				},
			},
		},
		{
			name: "test without allowed domain",
			allowedDomains: []string{
				"@dockyards.dev",
			},
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-without-allowed-domain",
				},
				Spec: dockyardsv1.UserSpec{
					Email: "test-without-allowed-domain@sudosweden.com",
				},
			},
			expected: apierrors.NewForbidden(
				dockyardsv1.GroupVersion.WithResource(dockyardsv1.UserKind).GroupResource(),
				"test-without-allowed-domain",
				field.Forbidden(
					field.NewPath("spec", "email"),
					"address is forbidden",
				),
			),
		},
		{
			name: "test invalid email address",
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-invalid-email-address",
				},
				Spec: dockyardsv1.UserSpec{
					Email: "test@invalid@email.address",
				},
			},
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.UserKind).GroupKind(),
				"test-invalid-email-address",
				field.ErrorList{
					field.Invalid(
						field.NewPath("spec", "email"),
						"test@invalid@email.address",
						"address is invalid",
					),
				},
			),
		},
		{
			name: "test with multiple allowed domains",
			allowedDomains: []string{
				"@dockyards.dev",
				"@sudosweden.com",
			},
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-multiple-allowed-domains",
				},
				Spec: dockyardsv1.UserSpec{
					Email: "test@dockyards.dev",
				},
			},
		},
		{
			name: "test without allowed domain but with voucher code",
			allowedDomains: []string{
				"@dockyards.dev",
			},
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-without-domain-with-voucher",
					Annotations: map[string]string{
						dockyardsv1.AnnotationVoucherCode: "ABC-123",
					},
				},
				Spec: dockyardsv1.UserSpec{
					Email: "test@sudosweden.com",
				},
			},
		},
		{
			name: "test existing user",
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: existingUser.ObjectMeta,
				Spec: dockyardsv1.UserSpec{
					DisplayName: "testing",
					Email:       existingUser.Spec.Email,
				},
			},
		},
		{
			name: "test existing email",
			dockyardsUser: dockyardsv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-existing-email",
				},
				Spec: dockyardsv1.UserSpec{
					Email: existingUser.Spec.Email,
				},
			},
			expected: apierrors.NewForbidden(
				dockyardsv1.GroupVersion.WithResource(dockyardsv1.UserKind).GroupResource(),
				"test-existing-email",
				field.Forbidden(
					field.NewPath("spec", "email"),
					"address is forbidden",
				),
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhook := webhooks.DockyardsUser{
				Client:         mgr.GetClient(),
				AllowedDomains: tc.allowedDomains,
			}

			_, actual := webhook.ValidateCreate(ctx, &tc.dockyardsUser)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
