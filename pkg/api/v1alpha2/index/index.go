package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MemberRefsIndexKey = ".spec.memberRefs.uid"
)

func MemberRefsIndexer(object client.Object) []string {
	organization := object.(*v1alpha2.Organization)

	memberUIDs := make([]string, len(organization.Spec.MemberRefs))
	for i, memberRef := range organization.Spec.MemberRefs {
		memberUIDs[i] = string(memberRef.UID)
	}

	return memberUIDs
}
