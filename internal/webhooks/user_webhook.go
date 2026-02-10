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

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=users,verbs=create;update,path=/validate-dockyards-io-v1alpha3-user,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.user.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

type DockyardsUser struct {
	Client client.Reader

	AllowedDomains []string
}

var _ admission.Validator[*dockyardsv1.User] = &DockyardsUser{}

func (webhook *DockyardsUser) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m, &dockyardsv1.User{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsUser) ValidateCreate(ctx context.Context, user *dockyardsv1.User) (admission.Warnings, error) {
	return nil, webhook.validate(ctx, user)
}

func (webhook *DockyardsUser) ValidateUpdate(ctx context.Context, _, newUser *dockyardsv1.User) (admission.Warnings, error) {
	return nil, webhook.validate(ctx, newUser)
}

func (webhook *DockyardsUser) ValidateDelete(_ context.Context, _ *dockyardsv1.User) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsUser) validate(ctx context.Context, dockyardsUser *dockyardsv1.User) error {
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

	matchingFields := client.MatchingFields{
		index.EmailField: dockyardsUser.Spec.Email,
	}

	var userList dockyardsv1.UserList
	err = webhook.Client.List(ctx, &userList, matchingFields)
	if err != nil {
		return err
	}

	for _, user := range userList.Items {
		if user.UID == dockyardsUser.UID {
			continue
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
