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
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/jsonrpc2"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/gorilla/websocket"
)

// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

type Method = string
const (
	EventObjectCreate     Method = "object/create"
	EventObjectUpdate     Method = "object/update"
	EventObjectDelete     Method = "object/delete"

	EventUserAuthenticate   Method = "user/authenticate"
	EventResourceWatch      Method = "resource/watch"
	EventResourceUnwatch    Method = "resource/unwatch"
)

type Event struct {
	Method Method
	Data  *unstructured.Unstructured
}

func (a *API) WebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(r.Context())

	ws, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("could not upgrade websocket connection", "err", err)
		return
	}
	defer ws.Close()

	writer := make(chan jsonrpc2.Message, 1024)
	defer close(writer)
	go func() {
		for message := range writer {
			bytes, err := json.Marshal(message)
			if err != nil {
				logger.Error("could not marshal message", "err", err)
				continue
			}
			err = ws.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Error("could not write message", "err", err)
				continue
			}
		}
	}()

	eventStream := make(chan Event, 1024)
	defer close(eventStream)
	var subject atomic.Pointer[string]
	go func() {
		for e := range eventStream {
			subj := subject.Load()
			if subj == nil {
				continue
			}
			gvk := e.Data.GetObjectKind().GroupVersionKind()

			restmapping, err := a.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				logger.Error("could not get resource rest mapping", "err", err)
				continue
			}

			resourceAttributes := authorizationv1.ResourceAttributes{
				Group:     gvk.Group,
				Name:      e.Data.GetName(),
				Namespace: e.Data.GetNamespace(),
				Resource:  restmapping.Resource.Resource,
				Verb:      "get",
				Version:   gvk.Version,
			}

			allowed, err := apiutil.IsSubjectAllowed(ctx, a, *subj, &resourceAttributes)
			if err != nil {
				logger.Error("could not check if user is allowed", "err", err)
				continue
			}

			if !allowed {
				continue
			}

			notification, err := jsonrpc2.NewNotificationWithBody(e.Method, e.Data)
			if err != nil {
				logger.Error("could not create notification", "err", err)
				continue
			}

			writer <- notification
		}
	}()

	type informerHandle struct {
		informer cache.Informer
		handle toolscache.ResourceEventHandlerRegistration
	}
	watchingResource := make(map[schema.GroupVersionKind]informerHandle)
	defer func() {
		for _, ih := range watchingResource {
			ih.informer.RemoveEventHandler(ih.handle)
		}
	}()

	for {
		_, payload, err := ws.ReadMessage()
		if err != nil {
			logger.Error("could not read message", "err", err)
			break
		}

		var message jsonrpc2.Message
		err = json.Unmarshal(payload, &message)
		if err != nil {
			logger.Error("could not parse message", "err", err)
			break
		}

		if message.Kind().IsResponseLike() {
			continue
		}

		switch message.Method {
		case EventUserAuthenticate:
			var token string
			err := json.Unmarshal(message.Params, &token)
			if err != nil {
				logger.Error("event could not parse event token", "err", err)
				writer <- jsonrpc2.NewError(message.ID, jsonrpc2.ErrInvalidParams)
				continue
			}
			s, err := a.subjectFromToken(token)
			if err != nil {
				logger.Error("could not get subject from token", "err", err)
				writer <- jsonrpc2.NewError(message.ID, jsonrpc2.ErrUnauthorized)
				continue
			}
			subject.Store(&s)
		case EventResourceUnwatch:
			var gvk [3]string
			err := json.Unmarshal(message.Params, &gvk)
			if err != nil {
				logger.Error("could not get group version kind", "err", err)
				writer <- jsonrpc2.NewError(message.ID, jsonrpc2.ErrInvalidParams)
				continue
			}
			groupVersionKind := schema.GroupVersionKind{
				Group: gvk[0],
				Version: gvk[1],
				Kind: gvk[2],
			}
			ih, ok := watchingResource[groupVersionKind]
			if !ok {
				continue
			}
			ih.informer.RemoveEventHandler(ih.handle)
			delete(watchingResource, groupVersionKind)
		case EventResourceWatch:
			var gvk [3]string
			err := json.Unmarshal(message.Params, &gvk)
			if err != nil {
				logger.Error("could not get group version kind", "err", err)
				writer <- jsonrpc2.NewError(message.ID, jsonrpc2.ErrInvalidParams)
				continue
			}

			groupVersionKind := schema.GroupVersionKind{
				Group: gvk[0],
				Version: gvk[1],
				Kind: gvk[2],
			}
			if _, ok := watchingResource[groupVersionKind]; ok {
				continue
			}

			var obj unstructured.Unstructured
			obj.SetGroupVersionKind(groupVersionKind)

			informer, err := a.cache.GetInformer(ctx, &obj, cache.BlockUntilSynced(false))
			if err != nil {
				logger.Error("could not get informer", "err", err)
				continue
			}

			handle, err := informer.AddEventHandlerWithResyncPeriod(toolscache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					eventStream <- Event{
						Method: EventObjectCreate,
						Data: obj.(*unstructured.Unstructured),
					}
				},
				UpdateFunc: func(_, newObj interface{}) {
					eventStream <- Event{
						Method: EventObjectUpdate,
						Data: newObj.(*unstructured.Unstructured),
					}
				},
				DeleteFunc: func(obj interface{}) {
					eventStream <- Event{
						Method: EventObjectDelete,
						Data: obj.(*unstructured.Unstructured),
					}
				},
			}, 30 * time.Second)
			if err != nil {
				logger.Error("could not add event handler to informer", "err", err)
				continue
			}

			watchingResource[groupVersionKind] = informerHandle{
				informer: informer,
				handle: handle,
			}
		default:
			logger.Info("unknown method", "method", message.Method)
			writer <- jsonrpc2.NewError(message.ID, jsonrpc2.ErrMethodNotFound)
		}
	}
}
