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
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *handler) CreateOrganizationInvitation(ctx context.Context, organization *dockyardsv1.Organization, request *types.InvitationOptions) (*types.Invitation, error) {
	invitation := dockyardsv1.Invitation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pending-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
		Spec: dockyardsv1.InvitationSpec{
			Email: request.Email,
			Role:  dockyardsv1.OrganizationMemberRole(request.Role),
		},
	}

	if request.Duration != nil {
		duration, err := time.ParseDuration(*request.Duration)
		if err != nil {
			return nil, err
		}

		invitation.Spec.Duration = &metav1.Duration{
			Duration: duration,
		}
	}

	err := h.Create(ctx, &invitation)
	if err != nil {
		return nil, err
	}

	response := types.Invitation{
		ID:        string(invitation.UID),
		Name:      invitation.Name,
		CreatedAt: invitation.CreationTimestamp.Time,
	}

	return &response, nil
}
