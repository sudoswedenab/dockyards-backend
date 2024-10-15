package webhooks

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/internal/feature"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:groups=dockyards.io,resources=nodepools,verbs=create;update,path=/validate-dockyards-io-v1alpha2-nodepool,mutating=false,failurePolicy=fail,sideEffects=none,admissionReviewVersions=v1,name=validation.nodepool.dockyards.io,versions=v1alpha2

type DockyardsNodePool struct{}

var _ webhook.CustomValidator = &DockyardsNodePool{}

func (webhook *DockyardsNodePool) SetupWebhookWithManager(m ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(m).
		For(&dockyardsv1.NodePool{}).
		WithValidator(webhook).
		Complete()
}

func (webhook *DockyardsNodePool) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsNodePool, ok := obj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(nil, dockyardsNodePool)
}

func (webhook *DockyardsNodePool) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldNodePool, ok := oldObj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	newNodePool, ok := newObj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(oldNodePool, newNodePool)
}

func (webhook *DockyardsNodePool) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *DockyardsNodePool) validate(oldNodePool, newNodePool *dockyardsv1.NodePool) error {
	var errorList field.ErrorList

	if newNodePool.Spec.Storage && !feature.IsEnabled(featurenames.FeatureStorageRole) {
		invalid := field.Invalid(field.NewPath("spec", "storage"), newNodePool.Spec.Storage, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if newNodePool.Spec.StorageResources != nil && !feature.IsEnabled(featurenames.FeatureStorageRole) {
		invalid := field.Invalid(field.NewPath("spec", "storageResources"), newNodePool.Spec.StorageResources, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if newNodePool.Spec.LoadBalancer && !feature.IsEnabled(featurenames.FeatureLoadBalancerRole) {
		invalid := field.Invalid(field.NewPath("spec", "loadBalancer"), newNodePool.Spec.LoadBalancer, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if oldNodePool != nil && feature.IsEnabled(featurenames.FeatureImmutableResources) {
		if !cmp.Equal(oldNodePool.Spec.Resources, newNodePool.Spec.Resources) {
			forbidden := field.Forbidden(field.NewPath("spec", "resources"), "immutable-resources feature is enabled")
			errorList = append(errorList, forbidden)
		}
	}

	names := make(map[string]bool)

	for i, storageResource := range newNodePool.Spec.StorageResources {
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

	if len(errorList) > 0 {
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind()

		return apierrors.NewInvalid(qualifiedKind, newNodePool.Name, errorList)
	}

	return nil
}
