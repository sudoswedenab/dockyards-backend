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

package v1alpha2

import (
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *Organization) ConvertTo(hub conversion.Hub) error {
	dst := hub.(*v1alpha3.Organization)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.DisplayName = src.Spec.DisplayName
	dst.Spec.Duration = src.Spec.Duration
	dst.Spec.SkipAutoAssign = src.Spec.SkipAutoAssign

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.ExpirationTimestamp = src.Status.ExpirationTimestamp
	dst.Status.ResourceQuotas = src.Status.ResourceQuotas

	if src.Status.NamespaceRef != "" {
		dst.Status.NamespaceRef = &corev1.LocalObjectReference{
			Name: src.Status.NamespaceRef,
		}
	}

	if len(src.Spec.MemberRefs) > 0 {
		dst.Spec.MemberRefs = make([]v1alpha3.OrganizationMemberReference, len(src.Spec.MemberRefs))

		for i, memberRef := range src.Spec.MemberRefs {
			dst.Spec.MemberRefs[i] = v1alpha3.OrganizationMemberReference{
				TypedLocalObjectReference: corev1.TypedLocalObjectReference{
					APIGroup: &memberRef.Group,
					Kind:     memberRef.Kind,
					Name:     memberRef.Name,
				},
				Role: v1alpha3.OrganizationMemberRole(memberRef.Role),
				UID:  memberRef.UID,
			}
		}
	}

	if src.Spec.Cloud.ProjectRef != nil {
		groupVersion, err := schema.ParseGroupVersion(src.Spec.Cloud.ProjectRef.APIVersion)
		if err != nil {
			return err
		}

		dst.Spec.ProjectRef = &corev1.TypedObjectReference{
			APIGroup:  &groupVersion.Group,
			Kind:      src.Spec.Cloud.ProjectRef.Kind,
			Name:      src.Spec.Cloud.ProjectRef.Name,
			Namespace: &src.Spec.Cloud.ProjectRef.Namespace,
		}
	}

	if src.Spec.Cloud.SecretRef != nil {
		dst.Spec.CredentialRef = &corev1.TypedObjectReference{
			Kind:      "Secret",
			Name:      src.Spec.Cloud.SecretRef.Name,
			Namespace: &src.Spec.Cloud.SecretRef.Namespace,
		}
	}

	return nil
}

func (dst *Organization) ConvertFrom(hub conversion.Hub) error {
	src := hub.(*v1alpha3.Organization)

	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.DisplayName = src.Spec.DisplayName
	dst.Spec.Duration = src.Spec.Duration
	dst.Spec.SkipAutoAssign = src.Spec.SkipAutoAssign

	dst.Status.Conditions = src.Status.Conditions
	dst.Status.ExpirationTimestamp = src.Status.ExpirationTimestamp
	dst.Status.ResourceQuotas = src.Status.ResourceQuotas

	if src.Status.NamespaceRef != nil {
		dst.Status.NamespaceRef = src.Status.NamespaceRef.Name
	}

	if len(src.Spec.MemberRefs) > 0 {
		dst.Spec.MemberRefs = make([]MemberReference, len(src.Spec.MemberRefs))

		for i, memberRef := range src.Spec.MemberRefs {
			dst.Spec.MemberRefs[i] = MemberReference{
				Kind: memberRef.Kind,
				Name: memberRef.Name,
				Role: MemberRole(memberRef.Role),
				UID:  memberRef.UID,
			}

			if memberRef.APIGroup != nil {
				dst.Spec.MemberRefs[i].Group = *memberRef.APIGroup
			}
		}
	}

	if src.Spec.ProjectRef != nil {
		dst.Spec.Cloud.ProjectRef = &NamespacedObjectReference{
			Kind: src.Spec.ProjectRef.Kind,
			Name: src.Spec.ProjectRef.Name,
		}

		if src.Spec.ProjectRef.APIGroup != nil {
			apiVersion := schema.GroupVersion{Group: *src.Spec.ProjectRef.APIGroup, Version: "v1alpha1"}

			dst.Spec.Cloud.ProjectRef.APIVersion = apiVersion.String()
		}

		if src.Spec.ProjectRef.Namespace != nil {
			dst.Spec.Cloud.ProjectRef.Namespace = *src.Spec.ProjectRef.Namespace
		}
	}

	if src.Spec.CredentialRef != nil {
		dst.Spec.Cloud.SecretRef = &NamespacedSecretReference{
			Name: src.Spec.CredentialRef.Name,
		}

		if src.Spec.CredentialRef.Namespace != nil {
			dst.Spec.Cloud.SecretRef.Namespace = *src.Spec.CredentialRef.Namespace
		}
	}

	return nil
}
