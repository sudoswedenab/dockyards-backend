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
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetWhoami(ctx context.Context) (*types.User, error) {
	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return nil, err
	}

	var user dockyardsv1.User
	err = h.Get(ctx, client.ObjectKey{Name: subject}, &user)
	if err != nil {
		return nil, err
	}

	response := types.User{
		CreatedAt: &user.CreationTimestamp.Time,
		Email:     user.Spec.Email,
		ID:        string(user.UID),
		Name:      user.Name,
	}

	readyCondition := meta.FindStatusCondition(user.Status.Conditions, dockyardsv1.ReadyCondition)
	if readyCondition != nil {
		response.UpdatedAt = &readyCondition.LastTransitionTime.Time
	}

	if user.Spec.ProviderID != nil {
		response.ProviderID = user.Spec.ProviderID
	}

	if len(user.Spec.DisplayName) > 0 {
		response.DisplayName = &user.Spec.DisplayName
	}

	return &response, nil
}
