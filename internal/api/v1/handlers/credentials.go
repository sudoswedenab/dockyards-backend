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
	"strings"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;delete;get;list;patch;watch
// +kubebuilder:rbac:groups=dockyards,resources=organizations,verbs=get;list;watch

func (h *handler) GetOrganizationCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

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
		Verb:     "get",
		Resource: "organizations",
		Group:    "dockyards.io",
		Name:     organization.Name,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to get organization", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var secretList corev1.SecretList
	err = h.List(ctx, &secretList, client.InNamespace(organization.Status.NamespaceRef.Name))
	if err != nil {
		logger.Error("error listing secrets", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	credentials := []types.Credential{}

	for _, secret := range secretList.Items {
		if secret.Type != dockyardsv1.SecretTypeCredential {
			continue
		}

		credential := types.Credential{
			ID:           string(secret.UID),
			Name:         strings.TrimPrefix(secret.Name, "credential-"),
			Organization: organization.Name,
		}

		credentialTemplate, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
		if has {
			credential.CredentialTemplate = &credentialTemplate
		}

		credentials = append(credentials, credential)
	}

	b, err := json.Marshal(&credentials)
	if err != nil {
		logger.Error("error marhalling credentials", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) CreateOrganizationCredential(ctx context.Context, organization *dockyardsv1.Organization, request *types.Credential) (*types.Credential, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-" + request.Name,
			Namespace: organization.Status.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.OrganizationKind,
					Name:       organization.Name,
					UID:        organization.UID,
				},
			},
		},
		Type: dockyardsv1.SecretTypeCredential,
	}

	if request.Data != nil {
		secret.Data = make(map[string][]byte)

		for key, value := range *request.Data {
			secret.Data[key] = value
		}
	}

	err := h.Create(ctx, &secret)
	if err != nil {
		return nil, err
	}

	credential := types.Credential{
		ID:   string(secret.UID),
		Name: secret.Name,
	}

	return &credential, err
}

func (h *handler) DeleteOrganizationCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	credentialName := r.PathValue("credentialName")
	if organizationName == "" || credentialName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("eror getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Verb:      "patch",
		Resource:  "clusters",
		Group:     "dockyards.io",
		Namespace: organization.Status.NamespaceRef.Name,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to patch clusters", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	objectKey := client.ObjectKey{
		Name:      "credential-" + credentialName,
		Namespace: organization.Status.NamespaceRef.Name,
	}

	var secret corev1.Secret
	err = h.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if secret.Type != dockyardsv1.SecretTypeCredential {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	err = h.Delete(ctx, &secret)
	if err != nil {
		logger.Error("error deleting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) GetOrganizationCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	credentialName := r.PathValue("credentialName")

	if credentialName == "" || organizationName == "" {
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
		Verb:     "get",
		Resource: "organizations",
		Group:    "dockyards.io",
		Name:     organization.Name,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to get organization", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	objectKey := client.ObjectKey{
		Name:      "credential-" + credentialName,
		Namespace: organization.Status.NamespaceRef.Name,
	}

	var secret corev1.Secret
	err = h.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if secret.Type != dockyardsv1.SecretTypeCredential {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	plaintextKeys := make(map[string]bool)

	v1Credential := types.Credential{
		ID:           string(secret.UID),
		Name:         strings.TrimPrefix(secret.Name, "credential-"),
		Organization: organization.Name,
	}

	credentialTemplateName, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
	if has {
		var credentialTemplate dockyardsv1.CredentialTemplate
		err := h.Get(ctx, client.ObjectKey{Name: credentialTemplateName, Namespace: h.namespace}, &credentialTemplate)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting credential template", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !apierrors.IsNotFound(err) {
			for _, option := range credentialTemplate.Spec.Options {
				if option.Plaintext {
					plaintextKeys[option.Key] = true
				}
			}
		}

		v1Credential.CredentialTemplate = &credentialTemplateName
	}

	if secret.Data != nil {
		data := make(map[string][]byte)

		for key, value := range secret.Data {
			if plaintextKeys[key] {
				data[key] = value

				continue
			}

			data[key] = nil
		}

		v1Credential.Data = &data
	}

	b, err := json.Marshal(&v1Credential)
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

func (h *handler) PutOrganizationCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	credentialName := r.PathValue("credentialName")

	if organizationName == "" || credentialName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	var organization dockyardsv1.Organization
	err = h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if organization.Status.NamespaceRef == nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Verb:      "patch",
		Resource:  "clusters",
		Group:     "dockyards.io",
		Namespace: organization.Status.NamespaceRef.Name,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to patch organization", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	objectKey := client.ObjectKey{
		Name:      "credential-" + credentialName,
		Namespace: organization.Status.NamespaceRef.Name,
	}

	var secret corev1.Secret
	err = h.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) || secret.Type != dockyardsv1.SecretTypeCredential {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("error reading request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	defer r.Body.Close()

	var credential types.Credential
	err = json.Unmarshal(body, &credential)
	if err != nil {
		logger.Error("error unmarshalling request body", "err", err, "body", body)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if credential.Data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	patch := client.MergeFrom(secret.DeepCopy())

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	for key, value := range *credential.Data {
		secret.Data[key] = value
	}

	err = h.Patch(ctx, &secret, patch)
	if err != nil {
		logger.Error("error patching secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	matchingFields := client.MatchingFields{
		index.MemberReferencesField: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error listing organizations", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	credentials := []types.Credential{}

	for _, organization := range organizationList.Items {
		if organization.Status.NamespaceRef == nil {
			continue
		}

		var secretList corev1.SecretList
		err := h.List(ctx, &secretList, client.InNamespace(organization.Status.NamespaceRef.Name))
		if err != nil {
			logger.Error("error listing secrets", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		for _, secret := range secretList.Items {
			if secret.Type != dockyardsv1.SecretTypeCredential {
				continue
			}

			credential := types.Credential{
				ID:           string(secret.UID),
				Name:         strings.TrimPrefix(secret.Name, "credential-"),
				Organization: organization.Name,
			}

			credentialTemplate, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
			if has {
				credential.CredentialTemplate = &credentialTemplate
			}

			credentials = append(credentials, credential)
		}
	}

	b, err := json.Marshal(&credentials)
	if err != nil {
		logger.Error("error marshalling credentials", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}
