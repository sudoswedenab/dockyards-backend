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
	"encoding/json"
	"net/http"

	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GetClusterResourceFunc[T any] func(context.Context, *dockyardsv1.Cluster, string) (*T, error)

func GetClusterResource[T any](h *handler, resource string, f GetClusterResourceFunc[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := middleware.LoggerFrom(ctx).With("resource", resource)

		organizationName := r.PathValue("organizationName")
		if organizationName == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		clusterName := r.PathValue("clusterName")
		if clusterName == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		resourceName := r.PathValue("resourceName")
		if resourceName == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		var organization dockyardsv1.Organization
		err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting organization", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if organization.Spec.NamespaceRef == nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		subject, err := middleware.SubjectFrom(ctx)
		if err != nil {
			logger.Error("error getting subject from context", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceAttributes := authorizationv1.ResourceAttributes{
			Group:     dockyardsv1.GroupVersion.Group,
			Namespace: organization.Spec.NamespaceRef.Name,
			Resource:  resource,
			Verb:      "get",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to get resource", "subject", subject, "organization", organization.Name)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		objectKey := client.ObjectKey{
			Name:      clusterName,
			Namespace: organization.Spec.NamespaceRef.Name,
		}

		var cluster dockyardsv1.Cluster
		err = h.Get(ctx, objectKey, &cluster)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting cluster", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		response, err := f(ctx, &cluster, resourceName)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		b, err := json.Marshal(response)
		if err != nil {
			logger.Error("error marshalling response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(b)
		if err != nil {
			logger.Error("error writing response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

type GetOrganizationResourceFunc[T any] func(context.Context, *dockyardsv1.Organization, string) (*T, error)

func GetOrganizationResource[T any](h *handler, resource string, f GetOrganizationResourceFunc[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := middleware.LoggerFrom(ctx).With("resource", resource)

		organizationName := r.PathValue("organizationName")
		if organizationName == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		resourceName := r.PathValue("resourceName")
		if resourceName == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		var organization dockyardsv1.Organization
		err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting organization", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if organization.Spec.NamespaceRef == nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		subject, err := middleware.SubjectFrom(ctx)
		if err != nil {
			logger.Error("error getting subject from context", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceAttributes := authorizationv1.ResourceAttributes{
			Group:     dockyardsv1.GroupVersion.Group,
			Namespace: organization.Spec.NamespaceRef.Name,
			Resource:  resource,
			Verb:      "get",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to get resource", "subject", subject, "organization", organization.Name)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		response, err := f(ctx, &organization, resourceName)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		b, err := json.Marshal(response)
		if err != nil {
			logger.Error("error marshalling response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(b)
		if err != nil {
			logger.Error("error writing response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

type GetGlobalResourceFunc[T any] func(context.Context, string) (*T, error)

func GetGlobalResource[T any](h *handler, resource string, f GetGlobalResourceFunc[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := middleware.LoggerFrom(ctx).With("resource", resource)

		resourceName := r.PathValue("resourceName")
		if resourceName == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		subject, err := middleware.SubjectFrom(ctx)
		if err != nil {
			logger.Error("error getting subject from context", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		resourceAttributes := authorizationv1.ResourceAttributes{
			Group:    dockyardsv1.GroupVersion.Group,
			Name:     resourceName,
			Resource: resource,
			Verb:     "get",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to get resource", "subject", subject)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		response, err := f(ctx, resourceName)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		b, err := json.Marshal(response)
		if err != nil {
			logger.Error("error marshalling response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(b)
		if err != nil {
			logger.Error("error writing response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}
