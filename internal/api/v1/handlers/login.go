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
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"golang.org/x/crypto/bcrypt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) CreateGlobalTokens(ctx context.Context, request *types.LoginOptions) (*types.Tokens, error) {
	matchingFields := client.MatchingFields{
		index.EmailField: request.Email,
	}

	var userList dockyardsv1.UserList
	err := h.List(ctx, &userList, matchingFields)
	if err != nil {
		return nil, err
	}

	if len(userList.Items) != 1 {
		return nil, apierrors.NewUnauthorized("unexpected users count")
	}

	user := userList.Items[0]

	condition := meta.FindStatusCondition(user.Status.Conditions, dockyardsv1.ReadyCondition)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return nil, apierrors.NewUnauthorized("user does not have ready condition")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(request.Password))
	if err != nil {
		return nil, apierrors.NewUnauthorized("error comparing password to hash")
	}

	tokens, err := h.generateTokens(&user)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	return tokens, nil
}
