package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
