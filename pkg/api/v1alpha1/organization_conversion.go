package v1alpha1

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *Organization) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.Organization)

	dst.ObjectMeta = src.ObjectMeta

	memberRefs := make([]v1alpha2.MemberReference, len(src.Spec.MemberRefs))
	for i, memberRef := range src.Spec.MemberRefs {
		memberRefs[i] = v1alpha2.MemberReference{
			Group: GroupVersion.Group,
			Kind:  UserKind,
			Name:  memberRef.Name,
			UID:   memberRef.UID,
			Role:  v1alpha2.MemberRoleSuperUser,
		}
	}

	dst.Spec.MemberRefs = memberRefs

	if src.Spec.CloudRef != nil {
		dst.Spec.Cloud.ProjectRef = &v1alpha2.NamespacedObjectReference{
			APIVersion: src.Spec.CloudRef.APIVersion,
			Kind:       src.Spec.CloudRef.Kind,
			Name:       src.Spec.CloudRef.Name,
			Namespace:  src.Spec.CloudRef.Namespace,
		}

		dst.Spec.Cloud.SecretRef = &v1alpha2.NamespacedSecretReference{
			Name:      src.Spec.CloudRef.SecretRef,
			Namespace: src.Status.NamespaceRef,
		}
	}

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.NamespaceRef = src.Status.NamespaceRef

	return nil
}

func (dst *Organization) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.Organization)

	dst.ObjectMeta = src.ObjectMeta

	memberRefs := []UserReference{}
	for _, memberRef := range src.Spec.MemberRefs {
		if memberRef.Role != v1alpha2.MemberRoleSuperUser {
			continue
		}

		memberRefs = append(memberRefs, UserReference{
			Name: memberRef.Name,
			UID:  memberRef.UID,
		})
	}

	if src.Spec.Cloud.ProjectRef != nil {
		dst.Spec.CloudRef = &CloudReference{
			APIVersion: src.Spec.Cloud.ProjectRef.APIVersion,
			Kind:       src.Spec.Cloud.ProjectRef.Kind,
			Name:       src.Spec.Cloud.ProjectRef.Name,
			Namespace:  src.Spec.Cloud.ProjectRef.Namespace,
		}

		if src.Spec.Cloud.SecretRef != nil {
			dst.Spec.CloudRef.SecretRef = src.Spec.Cloud.SecretRef.Name
		}
	}

	dst.Spec.MemberRefs = memberRefs

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.NamespaceRef = src.Status.NamespaceRef

	return nil
}
