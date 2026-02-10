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
	"fmt"
	"reflect"

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=members,verbs=create;update,path=/validate-dockyards-io-v1alpha3-member,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.member.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

// +kubebuilder:webhook:groups=dockyards.io,resources=members,verbs=create,path=/mutate-dockyards-io-v1alpha3-member,mutating=true,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=default.member.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

type DockyardsMember struct {
	Client client.Reader
}

var _ admission.Validator[*dockyardsv1.Member] = &DockyardsMember{}
var _ admission.Defaulter[*dockyardsv1.Member] = &DockyardsMember{}

func (webhook *DockyardsMember) Default(ctx context.Context, member *dockyardsv1.Member) error {
	organization, err := apiutil.GetOrganizationByNamespaceRef(ctx, webhook.Client, member.Namespace)
	if err != nil {
		return err
	}

	organizationName := organization.Name
	roleName := string(member.Spec.Role)
	userName := member.Spec.UserRef.Name

	if member.Labels == nil {
		member.Labels = make(map[string]string)
	}

	member.Labels[dockyardsv1.LabelOrganizationName] = organizationName
	member.Labels[dockyardsv1.LabelRoleName] = roleName
	member.Labels[dockyardsv1.LabelUserName] = userName

	return nil
}

func (webhook *DockyardsMember) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m, &dockyardsv1.Member{}).
		WithValidator(webhook).
		WithDefaulter(webhook).
		Complete()
}

func (webhook *DockyardsMember) ValidateCreate(ctx context.Context, member *dockyardsv1.Member) (admission.Warnings, error) {
	return webhook.validate(ctx, member)
}

func (webhook *DockyardsMember) ValidateUpdate(ctx context.Context, oldMember, newMember *dockyardsv1.Member) (admission.Warnings, error) {
	var errs field.ErrorList
	if !reflect.DeepEqual(oldMember.Spec.UserRef, newMember.Spec.UserRef) {
		invalid := field.Invalid(field.NewPath("spec", "userRef"), newMember.Spec.UserRef, "field is immutable")
		errs = append(errs, invalid)
	}

	if len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			dockyardsv1.GroupVersion.WithKind(dockyardsv1.MemberKind).GroupKind(),
			newMember.Name,
			errs,
		)
	}

	return webhook.validate(ctx, newMember)
}

func (webhook *DockyardsMember) ValidateDelete(_ context.Context, _ *dockyardsv1.Member) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsMember) validate(_ context.Context, newMember *dockyardsv1.Member) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errorList field.ErrorList

	if newMember.Spec.UserRef.Name == "" {
		invalid := field.Invalid(
			field.NewPath("spec", "userRef", "name"),
			newMember,
			"userRef must not be empty",
		)
		errorList = append(errorList, invalid)
	}

	if newMember.Spec.UserRef.Kind == "" {
		invalid := field.Invalid(
			field.NewPath("spec", "userRef", "kind"),
			newMember,
			fmt.Sprintf("invalid userRef kind, expected %s", dockyardsv1.UserKind),
		)
		errorList = append(errorList, invalid)
	}

	if newMember.Labels[dockyardsv1.LabelRoleName] != string(newMember.Spec.Role) {
		warnings = append(warnings, fmt.Sprintf("label '%s' should match the role defined in '%s'", dockyardsv1.LabelRoleName, field.NewPath("spec", "role")))
		newMember.Labels[dockyardsv1.LabelRoleName] = string(newMember.Spec.Role)
	}

	if newMember.Labels[dockyardsv1.LabelUserName] != string(newMember.Spec.UserRef.Name) {
		warnings = append(warnings, fmt.Sprintf("label '%s' should match the role defined in '%s'", dockyardsv1.LabelUserName, field.NewPath("spec", "userRef", "name")))
		newMember.Labels[dockyardsv1.LabelUserName] = string(newMember.Spec.UserRef.Name)
	}

	if len(errorList) > 0 {
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.MemberKind).GroupKind()

		return nil, apierrors.NewInvalid(qualifiedKind, newMember.Name, errorList)
	}

	return warnings, nil
}
