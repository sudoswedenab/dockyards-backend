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
	"k8s.io/utils/ptr"
)

// +kubebuilder:rbac:groups=dockyards,resources=organizations,verbs=get;list;watch

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
