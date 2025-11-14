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

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=members,verbs=delete;get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch

func (h *handler) ListOrganizationMembers(ctx context.Context, organization *dockyardsv1.Organization) (*[]types.Member, error) {
	var memberList dockyardsv1.MemberList
	err := h.List(ctx, &memberList, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return nil, err
	}

	response := make([]types.Member, len(memberList.Items))

	var user dockyardsv1.User
	for i, member := range memberList.Items {
		key := client.ObjectKey{
			Name: member.Spec.UserRef.Name,
		}

		err := h.Get(ctx, key, &user)
		if err != nil {
			return nil, err
		}

		response[i] = types.Member{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(user.UID),
			Name:      member.Name,
			Role:      ptr.To(string(member.Spec.Role)),
		}
	}

	return &response, nil
}

func (h *handler) DeleteOrganizationMember(ctx context.Context, organization *dockyardsv1.Organization, memberName string) error {
	key := client.ObjectKey{
		Name:      memberName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	if memberName == "@me" {
		subject, err := middleware.SubjectFrom(ctx)
		if err != nil {
			return err
		}

		key.Name = subject
	}

	var member dockyardsv1.Member
	err := h.Get(ctx, key, &member)
	if err != nil {
		return err
	}

	err = h.Delete(ctx, &member)
	if err != nil {
		return err
	}

	return nil
}
