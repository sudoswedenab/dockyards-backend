package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EmailIndexKey      = ".spec.email"
	UIDIndexKey        = ".metadata.uid"
	MemberRefsIndexKey = ".spec.memberRefs.uid"
	OwnerRefsIndexKey  = ".metadata.ownerReferences.uid"
)

func EmailIndexer(object client.Object) []string {
	return []string{
		object.(*v1alpha1.User).Spec.Email,
	}
}

func UIDIndexer(object client.Object) []string {
	return []string{
		string(object.GetUID()),
	}
}

func MemberRefsIndexer(object client.Object) []string {
	organization := object.(*v1alpha1.Organization)

	memberUIDs := make([]string, len(organization.Spec.MemberRefs))
	for i, memberRef := range organization.Spec.MemberRefs {
		memberUIDs[i] = string(memberRef.UID)
	}

	return memberUIDs
}

func OwnerRefsIndexer(object client.Object) []string {
	ownerReferences := object.GetOwnerReferences()

	ownerUIDs := make([]string, len(ownerReferences))
	for i, ownerReference := range ownerReferences {
		ownerUIDs[i] = string(ownerReference.UID)
	}

	return ownerUIDs
}
