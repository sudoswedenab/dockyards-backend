package apiutil

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOwnerOrganization(ctx context.Context, c client.Client, o client.Object) (*v1alpha2.Organization, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != v1alpha2.OrganizationKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group == v1alpha2.GroupVersion.Group {
			objectKey := client.ObjectKey{
				Name: ownerReference.Name,
			}

			var organization v1alpha2.Organization
			err := c.Get(ctx, objectKey, &organization)
			if err != nil {
				return nil, err
			}
			return &organization, nil
		}
	}

	return nil, nil
}

func GetOwnerCluster(ctx context.Context, c client.Client, o client.Object) (*v1alpha1.Cluster, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != v1alpha1.ClusterKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != v1alpha1.GroupVersion.Group {
			continue
		}

		if groupVersion.Group == v1alpha2.GroupVersion.Group {
			objectKey := client.ObjectKey{
				Name:      ownerReference.Name,
				Namespace: o.GetNamespace(),
			}

			var cluster v1alpha1.Cluster
			err := c.Get(ctx, objectKey, &cluster)
			if err != nil {
				return nil, err
			}
			return &cluster, nil
		}
	}

	return nil, nil
}

func GetOwnerNodePool(ctx context.Context, c client.Client, o client.Object) (*v1alpha1.NodePool, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != v1alpha1.NodePoolKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != v1alpha1.GroupVersion.Group {
			continue
		}

		objectKey := client.ObjectKey{
			Name:      ownerReference.Name,
			Namespace: o.GetNamespace(),
		}

		var nodePool v1alpha1.NodePool
		err = c.Get(ctx, objectKey, &nodePool)
		if err != nil {
			return nil, err
		}

		return &nodePool, nil
	}

	return nil, nil
}

func GetOwnerDeployment(ctx context.Context, c client.Client, o client.Object) (*v1alpha1.Deployment, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != v1alpha1.DeploymentKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != v1alpha1.GroupVersion.Group {
			continue
		}

		var deployment v1alpha1.Deployment
		err = c.Get(ctx, client.ObjectKeyFromObject(o), &deployment)
		if err != nil {
			return nil, err
		}

		return &deployment, nil
	}

	return nil, nil
}

func IsFeatureEnabled(ctx context.Context, c client.Client, featureName featurenames.FeatureName, namespace string) (bool, error) {
	var feature v1alpha1.Feature
	err := c.Get(ctx, client.ObjectKey{Name: string(featureName), Namespace: namespace}, &feature)
	if client.IgnoreNotFound(err) != nil {
		return false, err
	}

	if apierrors.IsNotFound(err) {
		return false, nil
	}

	return true, nil
}

func GetNamespaceOrganization(ctx context.Context, c client.Client, namespace string) (*v1alpha2.Organization, error) {
	var organizationList v1alpha2.OrganizationList
	err := c.List(ctx, &organizationList)
	if err != nil {
		return nil, err
	}

	for _, organization := range organizationList.Items {
		if organization.Status.NamespaceRef == namespace {
			return &organization, nil
		}
	}

	return nil, nil
}
