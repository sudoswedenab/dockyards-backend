package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EmailIndexKey            = ".spec.email"
	UIDIndexKey              = ".metadata.uid"
	MemberRefsIndexKey       = ".spec.memberRefs.uid"
	OwnerRefsIndexKey        = ".metadata.ownerReferences.uid"
	SecretTypeIndexKey       = ".type"
	ClusterServiceIDIndexKey = ".status.clusterServiceID"
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

func SecretTypeIndexer(object client.Object) []string {
	secret := object.(*corev1.Secret)

	return []string{
		string(secret.Type),
	}
}

func IndexByClusterServiceID(o client.Object) []string {
	switch t := o.(type) {
	case *v1alpha1.Cluster:
		return []string{t.Status.ClusterServiceID}
	case *v1alpha1.NodePool:
		return []string{t.Status.ClusterServiceID}
	case *v1alpha1.Node:
		return []string{t.Status.ClusterServiceID}
	default:
		return nil
	}
}
