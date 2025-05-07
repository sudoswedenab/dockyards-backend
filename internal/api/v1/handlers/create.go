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

type CreateClusterResourceFunc[T1, T2 any] func(context.Context, *dockyardsv1.Cluster, *T1) (*T2, error)

func CreateClusterResource[T1, T2 any](h *handler, resource string, f CreateClusterResourceFunc[T1, T2]) http.HandlerFunc {
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
			Verb:      "create",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to create resource", "subject", subject, "organization", organization.Name)
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

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("error reading request body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var request T1
		err = json.Unmarshal(b, &request)
		if err != nil {
			logger.Error("error unmarshalling request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		response, err := f(ctx, &cluster, &request)
		if apiutil.IgnoreClientError(err) != nil {
			logger.Error("error creating resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) {
			w.WriteHeader(http.StatusConflict)

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

		b, bytes := any(*response).([]byte)
		if !bytes {
			b, err = json.Marshal(response)
			if err != nil {
				logger.Error("error marshalling response", "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(b)
		if err != nil {
			logger.Error("error writing response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

type CreateOrganizationResourceFunc[T1, T2 any] func(context.Context, *dockyardsv1.Organization, *T1) (*T2, error)

func CreateOrganizationResource[T1, T2 any](h *handler, resource string, f CreateOrganizationResourceFunc[T1, T2]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := middleware.LoggerFrom(ctx).With("resource", resource)

		organizationName := r.PathValue("organizationName")
		if organizationName == "" {
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
			Verb:      "create",
		}

		allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
		if err != nil {
			logger.Error("error reviewing subject", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !allowed {
			logger.Debug("subject is not allowed to create resource", "subject", subject, "organization", organization.Name)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("error reading request body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var request T1
		err = json.Unmarshal(b, &request)
		if err != nil {
			logger.Error("error unmarshalling request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		response, err := f(ctx, &organization, &request)
		if apiutil.IgnoreClientError(err) != nil {
			logger.Error("error creating resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) {
			w.WriteHeader(http.StatusConflict)

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

		b, err = json.Marshal(response)
		if err != nil {
			logger.Error("error marshalling response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(b)
		if err != nil {
			logger.Error("error writing response", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

type CreateGlobalResourceFunc[T1, T2 any] func(context.Context, *T1) (*T2, error)

func CreateGlobalResource[T1, T2 any](resource string, f CreateGlobalResourceFunc[T1, T2]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := middleware.LoggerFrom(ctx).With("resource", resource)

		b, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("error reading request body", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		var request T1
		err = json.Unmarshal(b, &request)
		if err != nil {
			logger.Error("error unmarshalling request", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		response, err := f(ctx, &request)
		if apiutil.IgnoreIsInvalid(err) != nil {
			logger.Error("error creating global resource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

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

		b, err = json.Marshal(&response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusCreated)

		_, err = w.Write(b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}
