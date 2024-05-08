package webhooks

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/internal/feature"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
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

func (webhook *DockyardsNodePool) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dockyardsNodePool, ok := obj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsNodePool)
}

func (webhook *DockyardsNodePool) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	dockyardsNodePool, ok := newObj.(*dockyardsv1.NodePool)
	if !ok {
		return nil, nil
	}

	return nil, webhook.validate(dockyardsNodePool)
}

func (webhook *DockyardsNodePool) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (wehook *DockyardsNodePool) validate(dockyardsNodePool *dockyardsv1.NodePool) error {
	var errorList field.ErrorList

	if dockyardsNodePool.Spec.Storage && !feature.IsEnabled(featurenames.FeatureStorageRole) {
		invalid := field.Invalid(field.NewPath("spec", "storage"), dockyardsNodePool.Spec.Storage, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if dockyardsNodePool.Spec.StorageResources != nil && !feature.IsEnabled(featurenames.FeatureStorageRole) {
		invalid := field.Invalid(field.NewPath("spec", "storageResources"), dockyardsNodePool.Spec.StorageResources, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if dockyardsNodePool.Spec.LoadBalancer && !feature.IsEnabled(featurenames.FeatureLoadBalancerRole) {
		invalid := field.Invalid(field.NewPath("spec", "loadBalancer"), dockyardsNodePool.Spec.LoadBalancer, "feature is not enabled")
		errorList = append(errorList, invalid)
	}

	if len(errorList) > 0 {
		qualifiedKind := dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind()
		return apierrors.NewInvalid(qualifiedKind, dockyardsNodePool.Name, errorList)
	}

	return nil
}
