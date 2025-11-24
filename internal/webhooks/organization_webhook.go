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

package webhooks

import (
	"context"

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=organizations,verbs=create;update,path=/validate-dockyards-io-v1alpha3-organization,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.organization.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

// +kubebuilder:webhook:groups=dockyards.io,resources=organizations,verbs=create,path=/mutate-dockyards-io-v1alpha3-organization,mutating=true,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=default.organization.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

type DockyardsOrganization struct {
	Client client.Reader
}

var _ webhook.CustomValidator = &DockyardsOrganization{}
var _ webhook.CustomDefaulter = &DockyardsOrganization{}

func (webhook *DockyardsOrganization) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&dockyardsv1.Organization{}).
		WithValidator(webhook).
		WithDefaulter(webhook).
		Complete()
}

func (webhook *DockyardsOrganization) Default(_ context.Context, obj runtime.Object) error {
	organization, ok := obj.(*dockyardsv1.Organization)
	if !ok {
		return apierrors.NewBadRequest("new object has an unexpected type")
	}

	if organization.Labels == nil {
		organization.Labels = make(map[string]string)
	}

	organization.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

	return nil
}

func (webhook *DockyardsOrganization) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsOrganization, ok := obj.(*dockyardsv1.Organization)
	if !ok {
		return nil, nil
	}

	return webhook.validate(ctx, dockyardsOrganization)
}

func (webhook *DockyardsOrganization) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	dockyardsOrganization, ok := newObj.(*dockyardsv1.Organization)
	if !ok {
		return nil, nil
	}

	return webhook.validate(ctx, dockyardsOrganization)
}

func (webhook *DockyardsOrganization) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsOrganization) validate(ctx context.Context, dockyardsOrganization *dockyardsv1.Organization) (admission.Warnings, error) {
	var errorList field.ErrorList
	var warnings admission.Warnings

	organizationAutoAssignEnabled, err := apiutil.IsFeatureEnabled(ctx, webhook.Client, featurenames.FeatureOrganizationAutoAssign, corev1.NamespaceAll)
	if err != nil {
		return nil, err
	}

	if !dockyardsOrganization.Spec.SkipAutoAssign && !organizationAutoAssignEnabled {
		invalid := field.Invalid(field.NewPath("spec", "skipAutoAssign"), dockyardsOrganization.Spec.SkipAutoAssign, "feature is not enabled")

		errorList = append(errorList, invalid)
	}

	superUsers := 0
	for _, memberRef := range dockyardsOrganization.Spec.MemberRefs { //nolint:staticcheck
		if memberRef.Role == dockyardsv1.OrganizationMemberRoleSuperUser { //nolint:staticcheck
			superUsers = superUsers + 1
		}
	}

	if len(dockyardsOrganization.Spec.MemberRefs) > 0 { //nolint:staticcheck
		warnings = append(warnings, "spec.memberRefs is deprecated and will be removed in a future release; please migrate to using Member type instead.")
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

		return warnings, apierrors.NewInvalid(
			qualifiedKind,
			dockyardsOrganization.Name,
			errorList,
		)
	}

	return warnings, nil
}
