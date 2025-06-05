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

package handlers

import (
	"context"
	"slices"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards,resources=organizations,verbs=get;list;patch;watch

func (h *handler) ListOrganizationMembers(_ context.Context, organization *dockyardsv1.Organization) (*[]types.Member, error) {
	response := make([]types.Member, len(organization.Spec.MemberRefs))

	for i, memberRef := range organization.Spec.MemberRefs {
		response[i] = types.Member{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(memberRef.UID),
			Name:      memberRef.Name,
			Role:      ptr.To(string(memberRef.Role)),
		}
	}

	return &response, nil
}

func (h *handler) DeleteOrganizationMember(ctx context.Context, organization *dockyardsv1.Organization, memberName string) error {
	patch := client.MergeFrom(organization.DeepCopy())

	memberRefs := slices.DeleteFunc(organization.Spec.MemberRefs, func(memberRef dockyardsv1.OrganizationMemberReference) bool {
		return memberRef.Name == memberName
	})

	if slices.Equal(organization.Spec.MemberRefs, memberRefs) {
		return apierrors.NewNotFound(dockyardsv1.GroupVersion.WithResource("Member").GroupResource(), memberName)
	}

	organization.Spec.MemberRefs = memberRefs

	err := h.Patch(ctx, organization, patch)
	if err != nil {
		return err
	}

	return nil
}
