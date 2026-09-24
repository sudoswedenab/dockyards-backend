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
	"net/http"

	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (a *API) DeleteNamespacedResource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	subject, err := a.subjectFrom(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	gvr := schema.GroupVersionResource{
		Group: r.PathValue("group"),
		Version: r.PathValue("version"),
		Resource: r.PathValue("resource"),
	}

	gvk, err := a.RESTMapper().KindFor(gvr)
	if err != nil {
		applyError(w, err)
		logger.Error("could not get kind from gvr", "gvr", gvr, "err", err)

		return
	}

	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	var u unstructured.Unstructured
	u.SetGroupVersionKind(gvk)
	u.SetName(name)
	u.SetNamespace(namespace)

	if !a.isAllowed(ctx, subject, &u, "delete") {
		w.WriteHeader(http.StatusForbidden)

		return
	}

	err = a.Delete(ctx, &u)
	if err != nil {
		applyError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
