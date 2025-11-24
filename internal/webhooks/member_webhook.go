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
	"errors"
	"fmt"
	"reflect"

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=members,verbs=create;update,path=/validate-dockyards-io-v1alpha3-member,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.member.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

// +kubebuilder:webhook:groups=dockyards.io,resources=members,verbs=create,path=/mutate-dockyards-io-v1alpha3-member,mutating=true,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=default.member.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

type DockyardsMember struct {
	Client client.Reader
}

var _ webhook.CustomValidator = &DockyardsMember{}
var _ webhook.CustomDefaulter = &DockyardsMember{}

var memberLabels = []string{
	dockyardsv1.LabelOrganizationName,
	dockyardsv1.LabelUserName,
	dockyardsv1.LabelRoleName,
}

func (webhook *DockyardsMember) Default(ctx context.Context, obj runtime.Object) error {
	member, ok := obj.(*dockyardsv1.Member)
	if !ok {
		return apierrors.NewBadRequest("new object has an unexpected type")
	}

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
	return ctrl.NewWebhookManagedBy(m).
		For(&dockyardsv1.Member{}).
		WithValidator(webhook).
		WithDefaulter(webhook).
		Complete()
}

func (webhook *DockyardsMember) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsMember, ok := obj.(*dockyardsv1.Member)
	if !ok {
		return nil, apierrors.NewBadRequest("new object has an unexpected type")
	}

	return webhook.validate(ctx, dockyardsMember)
}

func (webhook *DockyardsMember) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newMember, ok := newObj.(*dockyardsv1.Member)
	if !ok {
		return nil, apierrors.NewBadRequest("new object has an unexpected type")
	}

	oldMember, ok := oldObj.(*dockyardsv1.Member)
	if !ok {
		return nil, apierrors.NewInternalError(errors.New("existing object has an unexpected type"))
	}

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

func (webhook *DockyardsMember) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsMember) validate(_ context.Context, newMember *dockyardsv1.Member) (admission.Warnings, error) {
	var errorList field.ErrorList

	for _, label := range memberLabels {
		if newMember.Labels[label] == "" {
			invalid := field.Invalid(
				field.NewPath("metadata", "labels"),
				newMember.Labels,
				fmt.Sprintf("missing value for label '%s'", label),
			)
			errorList = append(errorList, invalid)
		}
	}

	if newMember.Labels[dockyardsv1.LabelRoleName] != string(newMember.Spec.Role) {
		invalid := field.Invalid(
			field.NewPath("metadata", "labels"),
			newMember.Labels,
			fmt.Sprintf("label '%s' must match the role defined in '%s'", dockyardsv1.LabelRoleName, field.NewPath("spec", "role")),
		)
		errorList = append(errorList, invalid)
	}

	if newMember.Labels[dockyardsv1.LabelUserName] != string(newMember.Spec.UserRef.Name) {
		invalid := field.Invalid(
			field.NewPath("metadata", "labels"),
			newMember.Labels,
			fmt.Sprintf("label '%s' must match the name defined in '%s'", dockyardsv1.LabelUserName, field.NewPath("spec", "userRef", "name")),
		)
		errorList = append(errorList, invalid)
	}

	if len(errorList) > 0 {
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.MemberKind).GroupKind()

		return nil, apierrors.NewInvalid(qualifiedKind, newMember.Name, errorList)
	}

	return nil, nil
}
