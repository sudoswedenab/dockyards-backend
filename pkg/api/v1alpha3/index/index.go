package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EmailField            = ".spec.email"
	MemberReferencesField = ".spec.memberRefs.uid"
	OwnerReferencesField  = ".metadata.ownerReferences"
	SecretTypeField       = ".type"
	UIDField              = ".metadata.uid"
)

func ByMemberReferences(obj client.Object) []string {
	organization, ok := obj.(*v1alpha3.Organization)
	if !ok {
		return nil
	}

	memberUIDs := make([]string, len(organization.Spec.MemberRefs))
	for i, memberRef := range organization.Spec.MemberRefs {
		memberUIDs[i] = string(memberRef.UID)
	}

	return memberUIDs
}

func ByOwnerReferences(obj client.Object) []string {
	ownerReferences := obj.GetOwnerReferences()

	ownerUIDs := make([]string, len(ownerReferences))
	for i, ownerReference := range ownerReferences {
		ownerUIDs[i] = string(ownerReference.UID)
	}

	return ownerUIDs
}

func ByUID(obj client.Object) []string {
	return []string{string(obj.GetUID())}
}

func ByEmail(obj client.Object) []string {
	user, ok := obj.(*v1alpha3.User)
	if !ok {
		return nil
	}

	return []string{user.Spec.Email}
}

func BySecretType(obj client.Object) []string {
	secret := obj.(*corev1.Secret)

	return []string{
		string(secret.Type),
	}
}
