package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MemberRefsIndexKey   = ".spec.memberRefs.uid"
	CloudProjectRefKey   = ".spec.cloud.projectRef"
	EmailField           = ".spec.email"
	UIDField             = ".metadata.uid"
	OwnerReferencesField = ".metadata.ownerReferences"
	SecretTypeField      = ".type"
)

func MemberRefsIndexer(object client.Object) []string {
	organization := object.(*v1alpha2.Organization)

	memberUIDs := make([]string, len(organization.Spec.MemberRefs))
	for i, memberRef := range organization.Spec.MemberRefs {
		memberUIDs[i] = string(memberRef.UID)
	}

	return memberUIDs
}

func CloudRefValue(ref *v1alpha2.NamespacedObjectReference) string {
	return ref.Kind + "/" + ref.Namespace + "/" + ref.Name
}

func DockyardsOrganizationByCloudRef(o client.Object) []string {
	organization, ok := o.(*v1alpha2.Organization)
	if !ok {
		return nil
	}

	if organization.Spec.Cloud.ProjectRef == nil {
		return nil
	}

	return []string{CloudRefValue(organization.Spec.Cloud.ProjectRef)}
}

func ByEmail(o client.Object) []string {
	user, ok := o.(*v1alpha2.User)
	if !ok {
		return nil
	}

	return []string{user.Spec.Email}
}

func ByUID(o client.Object) []string {
	return []string{string(o.GetUID())}
}

func ByOwnerReferences(o client.Object) []string {
	ownerReferences := o.GetOwnerReferences()

	ownerUIDs := make([]string, len(ownerReferences))
	for i, ownerReference := range ownerReferences {
		ownerUIDs[i] = string(ownerReference.UID)
	}

	return ownerUIDs
}

func BySecretType(object client.Object) []string {
	secret := object.(*corev1.Secret)

	return []string{
		string(secret.Type),
	}
}
