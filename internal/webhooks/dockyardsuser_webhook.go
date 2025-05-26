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
	"net/mail"
	"strings"

	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=users,verbs=create;update,path=/validate-dockyards-io-v1alpha3-user,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.user.dockyards.io,versions=v1alpha3

type DockyardsUser struct {
	AllowedDomains []string
}

func (webhook *DockyardsUser) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m).
		For(&dockyardsv1.User{}).
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

	_, hasVoucherCode := dockyardsUser.Annotations[dockyardsv1.AnnotationVoucherCode]
	if hasVoucherCode {
		return nil
	}

	for _, allowedDomain := range webhook.AllowedDomains {
		if strings.HasSuffix(address.Address, allowedDomain) {
			return nil
		}
	}

	forbidden := field.Forbidden(
		field.NewPath("spec", "email"),
		"address is forbidden",
	)

	qualifiedResource := dockyardsv1.GroupVersion.WithResource(dockyardsv1.UserKind).GroupResource()

	return apierrors.NewForbidden(
		qualifiedResource,
		dockyardsUser.Name,
		forbidden,
	)
}
