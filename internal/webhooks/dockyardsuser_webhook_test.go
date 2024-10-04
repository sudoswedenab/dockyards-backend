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

func TestDockyardsUserValidateCreate(t *testing.T) {
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
			expected: apierrors.NewInvalid(
				dockyardsv1.GroupVersion.WithKind(dockyardsv1.UserKind).GroupKind(),
				"test-without-allowed-domain",
				field.ErrorList{
					field.Forbidden(
						field.NewPath("spec", "email"),
						"address is forbidden",
					),
				},
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			webhook := webhooks.DockyardsUser{
				AllowedDomains: tc.allowedDomains,
			}

			_, actual := webhook.ValidateCreate(context.Background(), &tc.dockyardsUser)
			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
