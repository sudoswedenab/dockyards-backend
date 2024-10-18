package webhooks

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/internal/feature"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=organizations,verbs=create;update,path=/validate-dockyards-io-v1alpha3-organization,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.organizations.dockyards.io,versions=v1alpha3

type DockyardsOrganization struct{}

var _ webhook.CustomValidator = &DockyardsNodePool{}

func (webhook *DockyardsOrganization) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&dockyardsv1.Organization{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsOrganization) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsOrganization, ok := obj.(*dockyardsv1.Organization)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsOrganization)
}

func (webhook *DockyardsOrganization) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	dockyardsOrganization, ok := newObj.(*dockyardsv1.Organization)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsOrganization)
}

func (webhook *DockyardsOrganization) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsOrganization) validate(dockyardsOrganization *dockyardsv1.Organization) error {
	var errorList field.ErrorList

	if !dockyardsOrganization.Spec.SkipAutoAssign && !feature.IsEnabled(featurenames.FeatureOrganizationAutoAssign) {
		invalid := field.Invalid(field.NewPath("spec", "skipAutoAssign"), dockyardsOrganization.Spec.SkipAutoAssign, "feature is not enabled")

		errorList = append(errorList, invalid)
	}

	superUsers := 0
	for _, memberRef := range dockyardsOrganization.Spec.MemberRefs {
		if memberRef.Role == dockyardsv1.OrganizationMemberRoleSuperUser {
			superUsers = superUsers + 1
		}
	}

	if superUsers < 1 {
		required := field.Required(
			field.NewPath("spec", "memberRefs"),
			"must have at least one super user",
		)

		errorList = append(errorList, required)
	}

	if len(errorList) > 0 {
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.OrganizationKind).GroupKind()

		return apierrors.NewInvalid(
			qualifiedKind,
			dockyardsOrganization.Name,
			errorList,
		)
	}

	return nil
}
