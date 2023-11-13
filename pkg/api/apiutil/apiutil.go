package apiutil

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
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
