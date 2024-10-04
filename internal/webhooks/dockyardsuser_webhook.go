package webhooks

import (
	"context"
	"net/mail"
	"strings"

	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=users,verbs=create;update,path=/validate-dockyards-io-v1alpha2-user,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.cluster.dockyards.io,versions=v1alpha2

type DockyardsUser struct {
	AllowedDomains []string
}

func (webhook *DockyardsUser) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m).
		For(&dockyardsv1.Cluster{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsUser) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsUser, ok := obj.(*dockyardsv1.User)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsUser)
}

func (webhook *DockyardsUser) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	dockyardsUser, ok := newObj.(*dockyardsv1.User)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsUser)
}

func (webhook *DockyardsUser) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsUser) validate(dockyardsUser *dockyardsv1.User) error {
	address, err := mail.ParseAddress(dockyardsUser.Spec.Email)
	if err != nil {
		invalid := field.Invalid(
			field.NewPath("spec", "email"),
			dockyardsUser.Spec.Email,
			"address is invalid",
		)

		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.UserKind).GroupKind()

		return apierrors.NewInvalid(
			qualifiedKind,
			dockyardsUser.Name,
			field.ErrorList{
				invalid,
			},
		)
	}

	if webhook.AllowedDomains == nil {
		return nil
	}

	for _, allowedDomain := range webhook.AllowedDomains {
		if !strings.HasSuffix(address.Address, allowedDomain) {
			forbidden := field.Forbidden(
				field.NewPath("spec", "email"),
				"address is forbidden",
			)

			qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.UserKind).GroupKind()

			return apierrors.NewInvalid(
				qualifiedKind,
				dockyardsUser.Name,
				field.ErrorList{
					forbidden,
				},
			)
		}
	}

	return nil
}
