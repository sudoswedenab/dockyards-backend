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

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *API) GetNamespacedResource(w http.ResponseWriter, r *http.Request) {
	subject, err := a.subjectFrom(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	ctx := r.Context()

	group := r.PathValue("group")
	version := r.PathValue("version")
	kind := r.PathValue("kind")

	key := client.ObjectKey{
		Name: kind + "." + group,
	}

	var customResourceDefinition apiextensionsv1.CustomResourceDefinition
	err = a.Get(ctx, key, &customResourceDefinition)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	groupVersionKind := schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    customResourceDefinition.Spec.Names.Kind,
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Group:     group,
		Name:      name,
		Namespace: namespace,
		Resource:  customResourceDefinition.Spec.Names.Plural,
		Verb:      "list",
		Version:   version,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, a, subject, &resourceAttributes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var u unstructured.Unstructured
	u.SetGroupVersionKind(groupVersionKind)

	key = client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	err = a.Get(ctx, key, &u)
	if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	b, err := json.Marshal(u.Object)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	_, err = w.Write(b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}
