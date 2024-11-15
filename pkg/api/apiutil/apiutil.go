package apiutil

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Expirable interface {
	GetExpiration() *metav1.Time
}

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

func IsFeatureEnabled(ctx context.Context, c client.Reader, featureName featurenames.FeatureName, inNamespace string) (bool, error) {
	var featureList dockyardsv1.FeatureList
	err := c.List(ctx, &featureList, client.InNamespace(inNamespace))
	if err != nil {
		return false, err
	}

	for _, feature := range featureList.Items {
		if feature.Name == string(featureName) {
			return true, nil
		}
	}

	return false, nil
}

func GetNamespaceOrganization(ctx context.Context, c client.Client, namespace string) (*dockyardsv1.Organization, error) {
	var organizationList dockyardsv1.OrganizationList
	err := c.List(ctx, &organizationList)
	if err != nil {
		return nil, err
	}

	for _, organization := range organizationList.Items {
		if organization.Status.NamespaceRef == nil {
			continue
		}

		if organization.Status.NamespaceRef.Name == namespace {
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

func IgnoreForbidden(err error) error {
	if apierrors.IsForbidden(err) {
		return nil
	}

	return err
}

func HasExpired(expirable Expirable) bool {
	expiration := expirable.GetExpiration()

	if expiration == nil {
		return false
	}

	if metav1.Now().After(expiration.Time) {
		return true
	}

	return false
}

func IgnoreInternalError(err error) error {
	if apierrors.IsInternalError(err) {
		return nil
	}

	return err
}

func GetDefaultRelease(ctx context.Context, c client.Client, releaseType dockyardsv1.ReleaseType) (*dockyardsv1.Release, error) {
	var releaseList dockyardsv1.ReleaseList
	err := c.List(ctx, &releaseList)
	if err != nil {
		return nil, err
	}

	for _, release := range releaseList.Items {
		if release.Spec.Type != releaseType {
			continue
		}

		_, isDefault := release.Annotations[dockyardsv1.AnnotationDefaultRelease]
		if isDefault {
			return &release, nil
		}
	}

	return nil, nil
}

func GetDefaultClusterTemplate(ctx context.Context, c client.Client) (*dockyardsv1.ClusterTemplate, error) {
	var clusterTemplateList dockyardsv1.ClusterTemplateList
	err := c.List(ctx, &clusterTemplateList)
	if err != nil {
		return nil, err
	}

	for _, clusterTemplate := range clusterTemplateList.Items {
		_, isDefault := clusterTemplate.Annotations[dockyardsv1.AnnotationDefaultTemplate]
		if isDefault {
			return &clusterTemplate, nil
		}
	}

	return nil, nil
}
