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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch

func (h *handler) PostRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		logger.Debug("empty or missing authorization header during refresh")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	refreshToken := strings.TrimPrefix(authorizationHeader, "Bearer ")

	// Parse the token string and a function for looking for the key.
	token, err := jwt.ParseWithClaims(refreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return h.jwtRefreshPublicKey, nil
	})
	if err != nil {
		logger.Error("error parsing token with claims", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		logger.Error("invalid token claims")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := claims.GetSubject()
	if err != nil {
		logger.Error("error getting subject from claims", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: subject,
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		logger.Error("error listing users", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if len(userList.Items) != 1 {
		logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	user := userList.Items[0]

	tokens, err := h.generateTokens(user)
	if err != nil {
		logger.Error("error generating tokens", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	b, err := json.Marshal(&tokens)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) generateTokens(user dockyardsv1.User) (*types.Tokens, error) {
	claims := jwt.RegisteredClaims{
		Subject:   string(user.UID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedAccessToken, err := token.SignedString(h.jwtAccessPrivateKey)
	if err != nil {
		return nil, err
	}

	refreshTokenClaims := jwt.RegisteredClaims{
		Subject:   string(user.UID),
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
