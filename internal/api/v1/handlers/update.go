// Copyright 2025 Sudo Sweden AB
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
	"io"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UpdateClusterResourceFunc[T any] func(context.Context, *dockyardsv1.Cluster, string, *T) error

func UpdateClusterResource[T any](h *handler, resource string, f UpdateClusterResourceFunc[T]) http.HandlerFunc {
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
			Verb:      "patch",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to patch resource", "subject", subject, "organization", organization.Name)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("error reading request body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

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

		var request T
		err = json.Unmarshal(b, &request)
		if err != nil {
			logger.Error("error unmarshalling request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		err = f(ctx, &cluster, resourceName, &request)
		if apierrors.IsForbidden(err) {
			w.WriteHeader(http.StatusForbidden)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if apierrors.IsInvalid(err) {
			statusError, ok := err.(*apierrors.StatusError)
			if !ok {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			if statusError.ErrStatus.Details == nil {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			var response types.UnprocessableEntityErrors

			for _, cause := range statusError.ErrStatus.Details.Causes {
				response.Errors = append(response.Errors, cause.Message)
			}

			b, err := json.Marshal(response)
			if err != nil {
				logger.Error("error marhalling response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			w.WriteHeader(http.StatusUnprocessableEntity)
			_, err = w.Write(b)
			if err != nil {
				logger.Error("error writing response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			return
		}

		if err != nil {
			logger.Error("error updating resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

type UpdateOrganizationResourceFunc[T any] func(context.Context, *dockyardsv1.Organization, string, *T) error

func UpdateOrganizationResource[T any](h *handler, resource string, f UpdateOrganizationResourceFunc[T]) http.HandlerFunc {
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
			Verb:      "patch",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to patch resource", "subject", subject, "organization", organization.Name)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("error reading request body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var request T
		err = json.Unmarshal(b, &request)
		if err != nil {
			logger.Error("error unmarshalling request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		err = f(ctx, &organization, resourceName, &request)
		if apierrors.IsForbidden(err) {
			w.WriteHeader(http.StatusForbidden)

			return
		}

		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if apierrors.IsInvalid(err) {
			statusError, ok := err.(*apierrors.StatusError)
			if !ok {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			if statusError.ErrStatus.Details == nil {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			var response types.UnprocessableEntityErrors

			for _, cause := range statusError.ErrStatus.Details.Causes {
				response.Errors = append(response.Errors, cause.Message)
			}

			b, err := json.Marshal(response)
			if err != nil {
				logger.Error("error marhalling response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			w.WriteHeader(http.StatusUnprocessableEntity)
			_, err = w.Write(b)
			if err != nil {
				logger.Error("error writing response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			return
		}

		if err != nil {
			logger.Error("error updating resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

type UpdateGlobalResourceFunc[T any] func(context.Context, string, *T) error

func UpdateGlobalResource[T any](h *handler, resource string, f UpdateGlobalResourceFunc[T]) http.HandlerFunc {
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
			Verb:     "patch",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to patch resource", "subject", subject, "resourceName", resourceName)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("error reading request body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var request T
		err = json.Unmarshal(b, &request)
		if err != nil {
			logger.Error("error unmarshalling request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		err = f(ctx, resourceName, &request)
		if apierrors.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if apierrors.IsInvalid(err) {
			statusError, ok := err.(*apierrors.StatusError)
			if !ok {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			if statusError.ErrStatus.Details == nil {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			var response types.UnprocessableEntityErrors

			for _, cause := range statusError.ErrStatus.Details.Causes {
				response.Errors = append(response.Errors, cause.Message)
			}

			b, err := json.Marshal(response)
			if err != nil {
				logger.Error("error marhalling response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			w.WriteHeader(http.StatusUnprocessableEntity)
			_, err = w.Write(b)
			if err != nil {
				logger.Error("error writing response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			return
		}

		if err != nil {
			logger.Error("error updating resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
