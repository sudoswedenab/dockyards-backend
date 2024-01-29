package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MemberRefsIndexKey = ".spec.memberRefs.uid"
	CloudProjectRefKey = ".spec.cloud.projectRef"
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

	return []string{CloudRefValue(organization.Spec.Cloud.ProjectRef)}
}
