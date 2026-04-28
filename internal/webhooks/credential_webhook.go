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

	corev1 "k8s.io/api/core/v1"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:,resources=secrets,verbs=create;update,path=/validate-dockyards-credential,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.credential.dockyards.io,serviceName=dockyards-backend

type DockyardsCredential struct {
	Client client.Reader
}

var _ admission.Validator[*corev1.Secret] = &DockyardsCredential{}

func (webhook *DockyardsCredential) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m, &corev1.Secret{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsCredential) ValidateCreate(ctx context.Context, o *corev1.Secret) (admission.Warnings, error) {
	return webhook.validate(ctx, o)
}

func (webhook *DockyardsCredential) ValidateUpdate(ctx context.Context, _, newObj *corev1.Secret) (admission.Warnings, error) {
	return webhook.validate(ctx, newObj)
}

func (webhook *DockyardsCredential) ValidateDelete(_ context.Context, _ *corev1.Secret) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsCredential) validate(_ context.Context, obj *corev1.Secret) (admission.Warnings, error) {
	if obj.Type != dockyardsv1.SecretTypeCredential {
		return nil, nil
	}

	var errs field.ErrorList

	owner, err := apiutil.FindOwnerReference(obj, dockyardsv1.OrganizationKind)
	if err != nil {
		errs = append(errs, field.Required(
			field.NewPath("metadata", "ownerReferences"),
			"secret does not have a organization owner",
		))
	}

	if value := obj.Labels[dockyardsv1.LabelOrganizationName]; value != owner.Name {
		errs = append(errs, field.Invalid(
			field.NewPath("metadata", "labels", dockyardsv1.LabelOrganizationName),
			value,
			fmt.Sprintf("expected '%s'", owner.Name),
		))
	}

	if len(errs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		dockyardsv1.GroupVersion.WithKind(dockyardsv1.MemberKind).GroupKind(),
		obj.Name,
		errs,
	)
}
