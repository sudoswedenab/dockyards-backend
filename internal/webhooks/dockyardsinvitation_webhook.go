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

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=invitations,verbs=create;update,path=/validate-dockyards-io-v1alpha3-invitation,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.invitation.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

type DockyardsInvitation struct {
Client client.Reader
}

var _ webhook.CustomValidator = &DockyardsInvitation{}

func (webhook *DockyardsInvitation) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&dockyardsv1.Invitation{}).
		WithValidator(webhook).
		Complete()
	if err != nil {
		return err
	}

	return nil
}

func (webhook *DockyardsInvitation) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	invitation, ok := obj.(*dockyardsv1.Invitation)
	if !ok {
		return nil, apierrors.NewBadRequest("unexpected type")
	}

	_, err := mail.ParseAddress(invitation.Spec.Email)
	if err != nil {
		invalid := field.Invalid(field.NewPath("spec", "email"), invitation.Spec.Email, "unable to parse as address")
		errs := field.ErrorList{invalid}

		return nil, apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.InvitationKind).GroupKind(), invitation.Name, errs)
	}

	matchingFields := client.MatchingFields{
		index.EmailField: invitation.Spec.Email,
	}

	var invitationList dockyardsv1.InvitationList
	err = webhook.Client.List(ctx, &invitationList, &matchingFields, client.InNamespace(invitation.Namespace))
	if err != nil {
		return nil, err
	}

	if len(invitationList.Items) != 0 {
		invalid := field.Invalid(field.NewPath("spec", "email"), invitation.Spec.Email, "address already invited")
		errs := field.ErrorList{invalid}

		return nil, apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.InvitationKind).GroupKind(), invitation.Name, errs)
	}

	var userList dockyardsv1.UserList
	err = webhook.Client.List(ctx, &userList, matchingFields)
	if err != nil {
		return nil, err
	}

	if len(userList.Items) == 0 {
		return nil, nil
	}

	existingUser := userList.Items[0]

	matchingFields = client.MatchingFields{
		index.MemberReferencesField: string(existingUser.UID),
	}

	var organizationList dockyardsv1.OrganizationList
	err = webhook.Client.List(ctx, &organizationList, matchingFields)
	if err != nil {
		return nil, err
	}

	for _, organization := range organizationList.Items {
		if organization.Spec.NamespaceRef == nil {
			continue
		}

		if organization.Spec.NamespaceRef.Name != invitation.Namespace {
			continue
		}

		for _, memberRef := range organization.Spec.MemberRefs {
			if memberRef.UID != existingUser.UID {
				continue
			}

			invalid := field.Invalid(field.NewPath("spec", "email"), invitation.Spec.Email, "user already member")
			errs := field.ErrorList{invalid}

			return nil, apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.InvitationKind).GroupKind(), invitation.Name, errs)
		}
	}

	return nil, nil
}

func (webhook *DockyardsInvitation) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsInvitation) ValidateUpdate(_ context.Context, _, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
