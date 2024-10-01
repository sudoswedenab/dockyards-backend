package apiutil

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOwnerOrganization(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.Organization, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.OrganizationKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group == dockyardsv1.GroupVersion.Group {
			objectKey := client.ObjectKey{
				Name: ownerReference.Name,
			}

			var organization dockyardsv1.Organization
			err := c.Get(ctx, objectKey, &organization)
			if err != nil {
				return nil, err
			}
			return &organization, nil
		}
	}

	return nil, nil
}

func GetOwnerCluster(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.Cluster, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.ClusterKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		if groupVersion.Group == dockyardsv1.GroupVersion.Group {
			objectKey := client.ObjectKey{
				Name:      ownerReference.Name,
				Namespace: o.GetNamespace(),
			}

			var cluster dockyardsv1.Cluster
			err := c.Get(ctx, objectKey, &cluster)
			if err != nil {
				return nil, err
			}
			return &cluster, nil
		}
	}

	return nil, nil
}

func GetOwnerNodePool(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.NodePool, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.NodePoolKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		objectKey := client.ObjectKey{
			Name:      ownerReference.Name,
			Namespace: o.GetNamespace(),
		}

		var nodePool dockyardsv1.NodePool
		err = c.Get(ctx, objectKey, &nodePool)
		if err != nil {
			return nil, err
		}

		return &nodePool, nil
	}

	return nil, nil
}

func GetOwnerDeployment(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.Deployment, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.DeploymentKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		var deployment dockyardsv1.Deployment
		err = c.Get(ctx, client.ObjectKeyFromObject(o), &deployment)
		if err != nil {
			return nil, err
		}

		return &deployment, nil
	}

	return nil, nil
}

func IsFeatureEnabled(ctx context.Context, c client.Client, featureName featurenames.FeatureName, namespace string) (bool, error) {
	var feature dockyardsv1.Feature
	err := c.Get(ctx, client.ObjectKey{Name: string(featureName), Namespace: namespace}, &feature)
	if client.IgnoreNotFound(err) != nil {
		return false, err
	}

	if apierrors.IsNotFound(err) {
		return false, nil
	}

	return true, nil
}

func GetNamespaceOrganization(ctx context.Context, c client.Client, namespace string) (*dockyardsv1.Organization, error) {
	var organizationList dockyardsv1.OrganizationList
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

func GetOwnerHelmDeployment(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.HelmDeployment, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.HelmDeploymentKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		var deployment dockyardsv1.HelmDeployment
		err = c.Get(ctx, client.ObjectKeyFromObject(o), &deployment)
		if err != nil {
			return nil, err
		}

		return &deployment, nil
	}

	return nil, nil
}

func GetOwnerKustomizeDeployment(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.KustomizeDeployment, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.KustomizeDeploymentKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		var deployment dockyardsv1.KustomizeDeployment
		err = c.Get(ctx, client.ObjectKeyFromObject(o), &deployment)
		if err != nil {
			return nil, err
		}

		return &deployment, nil
	}

	return nil, nil
}

func GetOwnerContainerImageDeployment(ctx context.Context, c client.Client, o client.Object) (*dockyardsv1.ContainerImageDeployment, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != dockyardsv1.ContainerImageDeploymentKind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return nil, err
		}

		if groupVersion.Group != dockyardsv1.GroupVersion.Group {
			continue
		}

		var deployment dockyardsv1.ContainerImageDeployment
		err = c.Get(ctx, client.ObjectKeyFromObject(o), &deployment)
		if err != nil {
			return nil, err
		}

		return &deployment, nil
	}

	return nil, nil
}

func IsSubjectAllowed(ctx context.Context, c client.Client, subject string, resourceAttributes *authorizationv1.ResourceAttributes) (bool, error) {
	accessReview := authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:               subject,
			ResourceAttributes: resourceAttributes,
		},
	}

	err := c.Create(ctx, &accessReview)
	if err != nil {
		return false, err
	}

	return accessReview.Status.Allowed, nil
}

func IgnoreConflict(err error) error {
	if apierrors.IsConflict(err) {
		return nil
	}

	return err
}
