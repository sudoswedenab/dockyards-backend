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

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/name"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=nodepools,verbs=create;update,path=/validate-dockyards-io-v1alpha3-nodepool,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.nodepool.dockyards.io,versions=v1alpha3,serviceName=dockyards-backend

type DockyardsNodePool struct {
	Client client.Reader
}

var _ webhook.CustomValidator = &DockyardsNodePool{}

var nodePoolLabels = []string{
	dockyardsv1.LabelOrganizationName,
	dockyardsv1.LabelClusterName,
}

func (webhook *DockyardsNodePool) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m).
		For(&dockyardsv1.NodePool{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsNodePool) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsNodePool, ok := obj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(ctx, nil, dockyardsNodePool)
}

func (webhook *DockyardsNodePool) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldNodePool, ok := oldObj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	newNodePool, ok := newObj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(ctx, oldNodePool, newNodePool)
}

func (webhook *DockyardsNodePool) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsNodePool) validate(ctx context.Context, oldNodePool, newNodePool *dockyardsv1.NodePool) error {
	var errorList field.ErrorList

	storageRoleEnabled, err := apiutil.IsFeatureEnabled(ctx, webhook.Client, featurenames.FeatureStorageRole, corev1.NamespaceAll)
	if err != nil {
		return err
	}

	if newNodePool.Spec.Storage && !storageRoleEnabled {
		invalid := field.Invalid(field.NewPath("spec", "storage"), newNodePool.Spec.Storage, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if newNodePool.Spec.StorageResources != nil && !storageRoleEnabled {
		invalid := field.Invalid(field.NewPath("spec", "storageResources"), newNodePool.Spec.StorageResources, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	loadBalancerRoleEnabled, err := apiutil.IsFeatureEnabled(ctx, webhook.Client, featurenames.FeatureLoadBalancerRole, corev1.NamespaceAll)
	if err != nil {
		return err
	}

	if newNodePool.Spec.LoadBalancer && !loadBalancerRoleEnabled {
		invalid := field.Invalid(field.NewPath("spec", "loadBalancer"), newNodePool.Spec.LoadBalancer, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	immutableResourcesEnabled, err := apiutil.IsFeatureEnabled(ctx, webhook.Client, featurenames.FeatureImmutableResources, corev1.NamespaceAll)
	if err != nil {
		return err
	}

	if oldNodePool != nil && immutableResourcesEnabled {
		if !cmp.Equal(oldNodePool.Spec.Resources, newNodePool.Spec.Resources) {
			forbidden := field.Forbidden(field.NewPath("spec", "resources"), "immutable-resources feature is enabled")
			errorList = append(errorList, forbidden)
		}

		if !cmp.Equal(oldNodePool.Spec.StorageResources, newNodePool.Spec.StorageResources) {
			forbidden := field.Forbidden(field.NewPath("spec", "storageResources"), "immutable-resources feature is enabled")
			errorList = append(errorList, forbidden)
		}
	}

	immutableStorageResourcesEnabled, err := apiutil.IsFeatureEnabled(ctx, webhook.Client, featurenames.FeatureImmutableStorageResources, corev1.NamespaceAll)
	if err != nil {
		return err
	}

	if oldNodePool != nil && immutableStorageResourcesEnabled {
		if !cmp.Equal(oldNodePool.Spec.StorageResources, newNodePool.Spec.StorageResources) {
			forbidden := field.Forbidden(field.NewPath("spec", "storageResources"), "immutable-storage-resources feature is enabled")
			errorList = append(errorList, forbidden)
		}
	}

	names := make(map[string]bool)

	hostPathEnabled, err := apiutil.IsFeatureEnabled(ctx, webhook.Client, featurenames.FeatureStorageResourceTypeHostPath, corev1.NamespaceAll)
	if err != nil {
		return err
	}

	for i, storageResource := range newNodePool.Spec.StorageResources {
		if storageResource.Type == dockyardsv1.StorageResourceTypeHostPath && !hostPathEnabled {
			invalid := field.Invalid(field.NewPath("spec", "storageResources").Index(i).Child("type"), storageResource.Type, "feature is not enabled")
			errorList = append(errorList, invalid)
		}

		_, validName := name.IsValidName(storageResource.Name)
		if !validName {
			invalid := field.Invalid(field.NewPath("spec", "storageResources").Index(i).Child("name"), storageResource.Name, "not a valid name")
			errorList = append(errorList, invalid)

			continue
		}

		_, duplicated := names[storageResource.Name]
		if duplicated {
			duplicate := field.Duplicate(field.NewPath("spec", "storageResources", "name"), storageResource.Name)
			errorList = append(errorList, duplicate)

			break
		}

		names[storageResource.Name] = true
	}

	if newNodePool.Spec.ControlPlane && newNodePool.Spec.Replicas != nil {
		if *newNodePool.Spec.Replicas == 0 {
			invalid := field.Invalid(field.NewPath("spec", "replicas"), *newNodePool.Spec.Replicas, "must be at least 1 for control plane")
			errorList = append(errorList, invalid)
		}
	}

	for _, label := range nodePoolLabels {
		if newNodePool.Labels[label] == "" {
			invalid := field.Invalid(
				field.NewPath("metadata", "labels"),
				newNodePool.Labels,
				fmt.Sprintf("missing value for label '%s'", label),
			)
			errorList = append(errorList, invalid)
		}
	}

	if len(errorList) > 0 {
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind()

		return apierrors.NewInvalid(qualifiedKind, newNodePool.Name, errorList)
	}

	return nil
}
