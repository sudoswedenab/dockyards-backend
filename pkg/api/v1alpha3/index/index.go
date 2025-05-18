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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EmailField                     = ".spec.email"
	MemberReferencesField          = ".spec.memberRefs.uid"
	OwnerReferencesField           = ".metadata.ownerReferences"
	SecretTypeField                = ".type"
	UIDField                       = ".metadata.uid"
	CredentialReferenceField       = ".spec.credentialRef"
	CodeField                      = ".spec.code"
	WorkloadTemplateReferenceField = ".spec.workloadTemplateRef"
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
	switch t := obj.(type) {
	case *v1alpha3.User:
		return []string{t.Spec.Email}
	case *v1alpha3.Invitation:
		return []string{t.Spec.Email}
	}

	return nil
}

func BySecretType(obj client.Object) []string {
	secret := obj.(*corev1.Secret)

	return []string{
		string(secret.Type),
	}
}

func TypedObjectRef(ref *corev1.TypedObjectReference) string {
	if ref.Namespace == nil {
		return ref.Kind + ref.Name
	}

	return *ref.Namespace + ref.Kind + ref.Name
}

func ByCredentialRef(obj client.Object) []string {
	organization, ok := obj.(*v1alpha3.Organization)
	if !ok {
		return nil
	}

	if organization.Spec.CredentialRef == nil {
		return nil
	}

	return []string{TypedObjectRef(organization.Spec.CredentialRef)}
}

func ByCode(obj client.Object) []string {
	organizationVoucher, ok := obj.(*v1alpha3.OrganizationVoucher)
	if !ok {
		return nil
	}

	if organizationVoucher.Spec.Code == "" {
		return nil
	}

	return []string{organizationVoucher.Spec.Code}
}

func ByWorkloadTemplateReference(obj client.Object) []string {
	workload, ok := obj.(*v1alpha3.Workload)
	if !ok {
		return nil
	}

	if workload.Spec.WorkloadTemplateRef == nil {
		return nil
	}

	return []string{TypedObjectRef(workload.Spec.WorkloadTemplateRef)}
}
