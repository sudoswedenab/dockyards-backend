// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package index

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Deprecated: use MemberReferencesField
	MemberRefsIndexKey = ".spec.memberRefs.uid"
	// Deprecated: use CloudProjectReferenceField
	CloudProjectRefKey = ".spec.cloud.projectRef"

	CloudProjectReferenceField = ".spec.cloud.projectRef"
	MemberReferencesField      = ".spec.memberRefs.uid"
	EmailField                 = ".spec.email"
	UIDField                   = ".metadata.uid"
	OwnerReferencesField       = ".metadata.ownerReferences"
	SecretTypeField            = ".type"
	CloudSecretReferenceField  = ".spec.cloud.secretRef"
	CodeField                  = ".spec.code"
)

// Deprecated: use ByMemberReferences
func MemberRefsIndexer(o client.Object) []string {
	return ByMemberReferences(o)
}

func ByMemberReferences(o client.Object) []string {
	organization := o.(*v1alpha2.Organization)

	memberUIDs := make([]string, len(organization.Spec.MemberRefs))
	for i, memberRef := range organization.Spec.MemberRefs {
		memberUIDs[i] = string(memberRef.UID)
	}

	return memberUIDs
}

func CloudRefValue(ref *v1alpha2.NamespacedObjectReference) string {
	return ref.Kind + "/" + ref.Namespace + "/" + ref.Name
}

// Deprecated: user ByCloudProjectReference
func DockyardsOrganizationByCloudRef(o client.Object) []string {
	return ByCloudProjectReference(o)
}

func ByCloudProjectReference(o client.Object) []string {
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

func BySecretType(o client.Object) []string {
	secret := o.(*corev1.Secret)

	return []string{
		string(secret.Type),
	}
}

func CloudSecretRef(ref *v1alpha2.NamespacedSecretReference) string {
	return ref.Namespace + ref.Name
}

func ByCloudSecretRef(obj client.Object) []string {
	organization, ok := obj.(*v1alpha2.Organization)
	if !ok {
		return nil
	}

	if organization.Spec.Cloud.SecretRef == nil {
		return nil
	}

	return []string{CloudSecretRef(organization.Spec.Cloud.SecretRef)}
}

func ByCode(obj client.Object) []string {
	organizationVoucher, ok := obj.(*v1alpha2.OrganizationVoucher)
	if !ok {
		return nil
	}

	if organizationVoucher.Spec.Code == "" {
		return nil
	}

	return []string{organizationVoucher.Spec.Code}
}
