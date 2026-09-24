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
	"context"
	"crypto"
	"crypto/ecdsa"
	"net/http"
	"strings"
	"sync"
	"io"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type API struct {
	client.Client
	*http.ServeMux

	upgrader websocket.Upgrader
	cache cache.Cache
	accessKey crypto.PublicKey
}

func NewAPI(mgr manager.Manager, accessKey crypto.PublicKey, allowedOrigins []string) *API {
	mux := http.NewServeMux()

	const bufferSize = 4096
	api := API{
		Client:    mgr.GetClient(),
		cache:     mgr.GetCache(),
		ServeMux:  mux,
		accessKey: accessKey,
		upgrader: websocket.Upgrader{
			EnableCompression: true,
			ReadBufferSize:  bufferSize,
			WriteBufferSize: bufferSize,
			WriteBufferPool: &sync.Pool{
				New: func() any {
					return make([]byte, 0, bufferSize)
				},
			},
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					// The client is not a web browser, so cors is irrelevant.
					return true
				}
				for _, o := range allowedOrigins {
					if origin == o {
						return true
					}
				}
				return false
			},
		},
	}

	return &api
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v2/group/{group}/version/{version}/kind/{kind}/namespace/{namespace}", a.ListNamespacedResource)
	mux.HandleFunc("GET /v2/group/{group}/version/{version}/kind/{kind}/namespace/{namespace}/name/{name}", a.GetNamespacedResource)
	mux.HandleFunc("DELETE /v2/group/{group}/version/{version}/resource/{resource}/namespace/{namespace}/name/{name}", a.DeleteNamespacedResource)
	mux.HandleFunc("/v2", a.HandleResource)
	mux.HandleFunc("/v2/ws", a.WebSocket)
}

func (a *API) subjectFromToken(tokenString string) (string, error) {
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

func (a *API) subjectFrom(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	return a.subjectFromToken(token)
}

func (a *API) isAllowed(ctx context.Context, subject string, resource client.Object, verb string) bool {
	gvk := resource.GetObjectKind().GroupVersionKind()

	restmapping, err := a.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Group:     gvk.Group,
		Name:      resource.GetName(),
		Namespace: resource.GetNamespace(),
		Resource:  restmapping.Resource.Resource,
		Verb:      verb,
		Version:   gvk.Version,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, a, subject, &resourceAttributes)
	if err != nil {
		return false
	}

	if !allowed {
		return false
	}

	return true
}

func applyError(w http.ResponseWriter, err error) {
	var status apierrors.APIStatus
	if !errors.As(err, &status) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s := status.Status()
	if s.Code == 0 {
		s.Code = 500
	}
	w.WriteHeader(int(s.Code))
	if s.Message == "" {
		return
	}
	_, _ = w.Write([]byte(s.Message))
}

func (a *API) HandleResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	switch r.Method {
	case "PUT":
	case "POST":
	case "PATCH":
	case "DELETE":
		break
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	subject, err := a.subjectFrom(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("could not read body", "err", err)

		return
	}

	var obj unstructured.Unstructured
	decoder := serializer.NewCodecFactory(a.Scheme()).UniversalDeserializer()
	_, _, err = decoder.Decode(body, nil, &obj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("could not unmarshal object", "err", err)

		return
	}

	switch r.Method {
	case "DELETE":
		if !a.isAllowed(ctx, subject, &obj, "delete") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err := a.Delete(ctx, &obj)
		if err != nil {
			logger.Error("could not delete object", "err", err)
			applyError(w, err)
			return
		}
	case "PUT":
		if !a.isAllowed(ctx, subject, &obj, "update") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err := a.Update(ctx, &obj)
		if err != nil {
			logger.Error("could not update object", "err", err)
			applyError(w, err)
			return
		}
	case "POST":
		if !a.isAllowed(ctx, subject, &obj, "create") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err := a.Create(ctx, &obj)
		if err != nil {
			applyError(w, err)
			logger.Error("could not create object", "err", err)
			return
		}
	case "PATCH":
		if !a.isAllowed(ctx, subject, &obj, "patch") {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err := a.Patch(ctx, &obj, client.StrategicMergeFrom(obj.DeepCopy()))
		if err != nil {
			applyError(w, err)
			logger.Error("could not patch object", "err", err)
			return
		}
	}

	b, err := obj.MarshalJSON()
	if err != nil {
		w.WriteHeader(http.StatusAccepted)
		logger.Error("could not marshal object", "err", err)

		return
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(b)
	if err != nil {
		return
	}
}
