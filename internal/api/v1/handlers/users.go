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
	"errors"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"golang.org/x/crypto/bcrypt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (h *handler) CreateGlobalUser(ctx context.Context, request *types.UserOptions) (*types.User, error) {
	enabled, err := apiutil.IsFeatureEnabled(ctx, h, featurenames.FeatureUserSignUp, h.namespace)
	if err != nil {
		return nil, err
	}

	if !enabled {
		err := errors.New("user sign-up feature is not enabled")

		return nil, apierrors.NewForbidden(dockyardsv1.GroupVersion.WithResource("users").GroupResource(), request.Email, err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockyards-",
		},
		Spec: dockyardsv1.UserSpec{
			Duration: &metav1.Duration{
				Duration: time.Hour * 12,
			},
			Email:      request.Email,
			Password:   string(passwordHash),
			ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
		},
	}

	if request.DisplayName != nil {
		user.Spec.DisplayName = *request.DisplayName
	}

	err = h.Create(ctx, &user)
	if err != nil {
		return nil, err
	}

	result := types.User{
		CreatedAt:  &user.CreationTimestamp.Time,
		Email:      user.Spec.Email,
		ID:         string(user.UID),
		Name:       user.Name,
		ProviderID: user.Spec.ProviderID,
	}

	if user.Spec.DisplayName != "" {
		result.DisplayName = &user.Spec.DisplayName
	}

	return &result, nil
}
