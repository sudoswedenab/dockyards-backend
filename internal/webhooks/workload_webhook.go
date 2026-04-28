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

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=workloads,verbs=create;delete;update,path=/validate-dockyards-io-v1alpha3-workload,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.workload.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend
// +kubebuilder:webhookconfiguration:mutating=false,name=dockyards-backend

// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

type DockyardsWorkload struct {
	Client client.Reader
}

var _ admission.Validator[*dockyardsv1.Workload] = &DockyardsWorkload{}

func (webhook *DockyardsWorkload) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m, &dockyardsv1.Workload{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsWorkload) ValidateCreate(_ context.Context, obj *dockyardsv1.Workload) (admission.Warnings, error) {
	return webhook.validate(obj)
}

func (webhook *DockyardsWorkload) ValidateDelete(_ context.Context, _ *dockyardsv1.Workload) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsWorkload) ValidateUpdate(_ context.Context, _, newObj *dockyardsv1.Workload) (admission.Warnings, error) {
	return webhook.validate(newObj)
}

func (webhook *DockyardsWorkload) validate(o *dockyardsv1.Workload) (admission.Warnings, error) {
	var errs field.ErrorList

	labelsField := field.NewPath("metadata", "labels")

	if o.Labels[dockyardsv1.LabelOrganizationName] == "" {
		errs = append(errs, field.Required(labelsField.Key(dockyardsv1.LabelOrganizationName), ""))
	}

	if o.Labels[dockyardsv1.LabelClusterName] == "" {
		errs = append(errs, field.Required(labelsField.Key(dockyardsv1.LabelClusterName), ""))
	}

	if o.Labels[dockyardsv1.LabelWorkloadName] == "" {
		errs = append(errs, field.Required(labelsField.Key(dockyardsv1.LabelWorkloadName), ""))
	}

	if o.Labels[dockyardsv1.LabelWorkloadTemplateName] == "" {
		errs = append(errs, field.Required(labelsField.Key(dockyardsv1.LabelWorkloadTemplateName), ""))
	}

	owner, err := apiutil.FindOwnerReference(o, dockyardsv1.ClusterKind)
	if err != nil {
		errs = append(errs, field.InternalError(field.NewPath("metadata", "ownerReferences"), err))
	}

	if value := o.Labels[dockyardsv1.LabelClusterName]; value != owner.Name {
		errs = append(errs, field.Invalid(
			labelsField.Key(dockyardsv1.LabelClusterName),
			value,
			fmt.Sprintf("expected '%s'", owner.Name),
		))
	}

	if value := o.Labels[dockyardsv1.LabelWorkloadTemplateName]; value != o.Spec.WorkloadTemplateRef.Name {
		errs = append(errs, field.Invalid(
			labelsField.Key(dockyardsv1.LabelWorkloadTemplateName),
			value,
			fmt.Sprintf("expected '%s'", o.Spec.WorkloadTemplateRef.Name),
		))
	}

	if len(errs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		dockyardsv1.GroupVersion.WithKind(dockyardsv1.WorkloadKind).GroupKind(),
		o.Name,
		errs,
	)
}
