package openstack

import (
	"context"
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	openstackv1alpha1 "bitbucket.org/sudosweden/dockyards-openstack/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=cloud.dockyards.io,resources=openstackprojects,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

var (
	ErrNoCloudReference  = errors.New("organization has no cloud reference")
	ErrNoOpenstackKind   = errors.New("organization does not have openstack kind")
	ErrNoSecretReference = errors.New("organization has no secret reference")
)

func (s *openStackService) getOpenstackProject(organization *v1alpha1.Organization) (*openstackv1alpha1.OpenstackProject, error) {
	if organization.Spec.CloudRef == nil {
		return nil, ErrNoCloudReference
	}

	if organization.Spec.CloudRef.Kind != openstackv1alpha1.OpenstackProjectKind {
		return nil, ErrNoOpenstackKind
	}

	ctx := context.Background()

	objectKey := client.ObjectKey{
		Name:      organization.Spec.CloudRef.Name,
		Namespace: organization.Spec.CloudRef.Namespace,
	}

	var openstackProject openstackv1alpha1.OpenstackProject
	err := s.controllerClient.Get(ctx, objectKey, &openstackProject)
	if err != nil {
		return nil, err
	}

	return &openstackProject, nil
}

func (s *openStackService) getOpenstackSecret(organization *v1alpha1.Organization) (*corev1.Secret, error) {
	if organization.Spec.CloudRef.SecretRef == "" {
		return nil, ErrNoSecretReference
	}

	ctx := context.Background()

	objectKey := client.ObjectKey{
		Name:      organization.Spec.CloudRef.SecretRef,
		Namespace: organization.Status.NamespaceRef,
	}

	var secret corev1.Secret
	err := s.controllerClient.Get(ctx, objectKey, &secret)
	if err != nil {
		return nil, err
	}

	return &secret, nil
}
