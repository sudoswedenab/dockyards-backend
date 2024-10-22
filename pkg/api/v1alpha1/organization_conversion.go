package v1alpha1

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *Organization) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha3.Organization)

	dst.ObjectMeta = src.ObjectMeta

	memberRefs := make([]v1alpha3.OrganizationMemberReference, len(src.Spec.MemberRefs))
	for i, memberRef := range src.Spec.MemberRefs {
		memberRefs[i] = v1alpha3.OrganizationMemberReference{
			TypedLocalObjectReference: corev1.TypedLocalObjectReference{
				APIGroup: &GroupVersion.Group,
				Kind:     UserKind,
				Name:     memberRef.Name,
			},
			UID:  memberRef.UID,
			Role: v1alpha3.OrganizationMemberRoleSuperUser,
		}
	}

	dst.Spec.MemberRefs = memberRefs

	if src.Spec.CloudRef != nil {
		groupVersion, err := schema.ParseGroupVersion(src.Spec.CloudRef.APIVersion)
		if err != nil {
			return err
		}

		dst.Spec.ProjectRef = &corev1.TypedObjectReference{
			APIGroup: &groupVersion.Group,
			Kind:     src.Spec.CloudRef.Kind,
			Name:     src.Spec.CloudRef.Name,
		}

		if src.Spec.CloudRef.Namespace != "" {
			dst.Spec.ProjectRef.Namespace = &src.Spec.CloudRef.Namespace
		}

		if src.Spec.CloudRef.SecretRef != "" {
			dst.Spec.CredentialRef = &corev1.TypedObjectReference{
				Name: src.Spec.CloudRef.SecretRef,
			}

			if src.Spec.CloudRef.Namespace != "" {
				dst.Spec.CredentialRef.Namespace = &src.Spec.CloudRef.Namespace
			}
		}
	}

	dst.Status.Conditions = src.Status.Conditions

	if src.Status.NamespaceRef != "" {
		dst.Status.NamespaceRef = &corev1.LocalObjectReference{
			Name: src.Status.NamespaceRef,
		}
	}

	return nil
}

func (dst *Organization) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha3.Organization)

	dst.ObjectMeta = src.ObjectMeta

	memberRefs := []UserReference{}
	for _, memberRef := range src.Spec.MemberRefs {
		if memberRef.Role != v1alpha3.OrganizationMemberRoleSuperUser {
			continue
		}

		memberRefs = append(memberRefs, UserReference{
			Name: memberRef.Name,
			UID:  memberRef.UID,
		})
	}

	if src.Spec.ProjectRef != nil {
		dst.Spec.CloudRef = &CloudReference{
			Kind: src.Spec.ProjectRef.Kind,
			Name: src.Spec.ProjectRef.Name,
		}

		if src.Spec.ProjectRef.APIGroup != nil {
			apiVersion := schema.GroupVersion{Group: *src.Spec.ProjectRef.APIGroup, Version: "v1alpha1"}

			dst.Spec.CloudRef.APIVersion = apiVersion.String()

		}

		if src.Spec.ProjectRef.Namespace != nil {
			dst.Spec.CloudRef.Namespace = *src.Spec.ProjectRef.Namespace
		}

	}

	dst.Spec.MemberRefs = memberRefs

	dst.Status.Conditions = src.Status.Conditions

	if src.Status.NamespaceRef != nil {
		dst.Status.NamespaceRef = src.Status.NamespaceRef.Name
	}

	return nil
}
