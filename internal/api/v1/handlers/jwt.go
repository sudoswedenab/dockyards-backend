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

	"github.com/golang-jwt/jwt/v5"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch

func (h *handler) GetGlobalTokens(ctx context.Context) (*types.Tokens, error) {
	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return nil, err
	}

	var user dockyardsv1.User
	err = h.Get(ctx, client.ObjectKey{Name: subject}, &user)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewUnauthorized("user not found")
		}

		return nil, err
	}

	response, err := h.generateTokens(&user)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (h *handler) generateTokens(user *dockyardsv1.User) (*types.Tokens, error) {
	claims := jwt.RegisteredClaims{
		Subject:   user.Name,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedAccessToken, err := token.SignedString(h.jwtAccessPrivateKey)
	if err != nil {
		return nil, err
	}

	refreshTokenClaims := jwt.RegisteredClaims{
		Subject:   user.Name,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2)),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodES256, refreshTokenClaims)
	signedRefreshToken, err := refreshToken.SignedString(h.jwtRefreshPrivateKey)
	if err != nil {
		return nil, err
	}

	tokens := types.Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	return &tokens, nil
}
