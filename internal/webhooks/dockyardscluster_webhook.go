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
	"net/netip"

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=clusters,verbs=create;delete;update,path=/validate-dockyards-io-v1alpha3-cluster,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.cluster.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend
// +kubebuilder:webhookconfiguration:mutating=false,name=dockyards-backend

type DockyardsCluster struct{}

func (webhook *DockyardsCluster) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m).
		For(&dockyardsv1.Cluster{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsCluster) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsCluster, ok := obj.(*dockyardsv1.Cluster)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsCluster)
}

func (webhook *DockyardsCluster) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsCluster, ok := obj.(*dockyardsv1.Cluster)
	if !ok {
		return nil, nil
	}

	if apiutil.HasExpired(dockyardsCluster) {
		return nil, nil
	}

	if dockyardsCluster.Spec.BlockDeletion {
		forbidden := field.Forbidden(
			field.NewPath("spec", "blockDeletion"),
			"deletion is blocked",
		)

		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind()

		return nil, apierrors.NewInvalid(
			qualifiedKind,
			dockyardsCluster.Name,
			field.ErrorList{
				forbidden,
			},
		)
	}

	return nil, nil
}

func (webhook *DockyardsCluster) ValidateUpdate(_ context.Context, oldObj runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	oldCluster, ok := oldObj.(*dockyardsv1.Cluster)
	if !ok {
		return nil, nil
	}

	newCluster, ok := newObj.(*dockyardsv1.Cluster)
	if !ok {
		return nil, nil
	}

	if newCluster.Spec.AllocateInternalIP != oldCluster.Spec.AllocateInternalIP {
		invalid := field.Invalid(
			field.NewPath("spec", "allocateInternalIP"),
			newCluster.Spec.AllocateInternalIP,
			"field is immutable",
		)

		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind()

		return nil, apierrors.NewInvalid(
			qualifiedKind,
			newCluster.Name,
			field.ErrorList{
				invalid,
			},
		)
	}

	return nil, webhook.validate(newCluster)
}

func (webhook *DockyardsCluster) validate(dockyardsCluster *dockyardsv1.Cluster) error {
	hasOrganizationOwner := false
	for _, ownerReference := range dockyardsCluster.OwnerReferences {
		if ownerReference.Kind != dockyardsv1.OrganizationKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		hasOrganizationOwner = true
	}

	if !hasOrganizationOwner {
		required := field.Required(
			field.NewPath("metadata", "ownerReferences"),
			"must have organization owner reference",
		)
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind()

		return apierrors.NewInvalid(
			qualifiedKind,
			dockyardsCluster.Name,
			field.ErrorList{
				required,
			},
		)
	}

	errorList := field.ErrorList{}

	for i, subnet := range dockyardsCluster.Spec.PodSubnets {
		_, err := netip.ParsePrefix(subnet)
		if err != nil {
			invalid := field.Invalid(field.NewPath("spec", "podSubnets").Index(i), subnet, "unable to parse pod subnet as prefix")
			errorList = append(errorList, invalid)
		}
	}

	for i, subnet := range dockyardsCluster.Spec.ServiceSubnets {
		_, err := netip.ParsePrefix(subnet)
		if err != nil {
			invalid := field.Invalid(field.NewPath("spec", "serviceSubnets").Index(i), subnet, "unable to parse service subnet as prefix")
			errorList = append(errorList, invalid)
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind()

	return apierrors.NewInvalid(
		qualifiedKind,
		dockyardsCluster.Name,
		errorList,
	)
}
