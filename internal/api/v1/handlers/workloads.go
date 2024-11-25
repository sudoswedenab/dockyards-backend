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
	"io"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=workloads,verbs=create;delete;get;list;watch

func (h *handler) CreateClusterWorkload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

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

	if organization.Status.NamespaceRef == nil {
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
		Namespace: organization.Status.NamespaceRef.Name,
		Resource:  "workloads",
		Verb:      "create",
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to create workloads", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	objectKey := client.ObjectKey{
		Name:      clusterName,
		Namespace: organization.Status.NamespaceRef.Name,
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

	var request types.Workload
	err = json.Unmarshal(b, &request)
	if err != nil {
		logger.Error("error unmarshalling request", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if request.WorkloadTemplateName == nil || request.Namespace == nil {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	workload := dockyardsv1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + *request.Name,
			Namespace: organization.Status.NamespaceRef.Name,
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
		},
		Spec: dockyardsv1.WorkloadSpec{
			Provenience:     dockyardsv1.ProvenienceUser,
			TargetNamespace: *request.Namespace,
			WorkloadTemplateRef: &corev1.TypedObjectReference{
				Kind:      dockyardsv1.WorkloadTemplateKind,
				Name:      *request.WorkloadTemplateName,
				Namespace: &h.namespace,
			},
		},
	}

	if request.Input != nil {
		raw, err := json.Marshal(*request.Input)
		if err != nil {
			logger.Error("error marshalling request input", "err", err)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		workload.Spec.Input = &apiextensionsv1.JSON{
			Raw: raw,
		}
	}

	err = h.Create(ctx, &workload)
	if apiutil.IgnoreClientError(err) != nil {
		logger.Error("error creating workload", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsConflict(err) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if apierrors.IsInvalid(err) {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	response := types.Workload{
		ID:   ptr.To(string(workload.UID)),
		Name: ptr.To(workload.Name),
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

func (h *handler) DeleteClusterWorkload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

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

	workloadName := r.PathValue("workloadName")
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

	if organization.Status.NamespaceRef == nil {
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
		Namespace: organization.Status.NamespaceRef.Name,
		Resource:  "workloads",
		Verb:      "delete",
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to delete workloads", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	objectKey := client.ObjectKey{
		Name:      clusterName + "-" + workloadName,
		Namespace: organization.Status.NamespaceRef.Name,
	}

	var workload dockyardsv1.Workload
	err = h.Get(ctx, objectKey, &workload)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting workload", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	err = h.Delete(ctx, &workload)
	if err != nil {
		logger.Error("error deleting workload", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusAccepted)
}
