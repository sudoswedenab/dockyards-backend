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

package v2

import (
	"crypto"
	"crypto/ecdsa"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type API struct {
	client.Client
	*http.ServeMux

	accessKey crypto.PublicKey
}

func NewAPI(mgr manager.Manager, accessKey crypto.PublicKey) *API {
	mux := http.NewServeMux()

	api := API{
		Client:    mgr.GetClient(),
		ServeMux:  mux,
		accessKey: accessKey,
	}

	return &api
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v2/group/{group}/version/{version}/kind/{kind}/namespace/{namespace}", a.ListNamespacedResource)
	mux.HandleFunc("/v2/group/{group}/version/{version}/kind/{kind}/namespace/{namespace}/name/{name}", a.GetNamespacedResource)
}

func (a *API) subjectFrom(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	tokenString := strings.TrimPrefix(header, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		_, ok := t.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}

		return a.accessKey.(*ecdsa.PublicKey), nil
	})
	if err != nil {
		return "", err
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		return "", err
	}

	return subject, nil
}
