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
	"strings"

	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;delete;get;list;patch;watch
// +kubebuilder:rbac:groups=dockyards,resources=organizations,verbs=get;list;watch

func (h *handler) ListOrganizationCredentials(ctx context.Context, organization *dockyardsv1.Organization) (*[]types.Credential, error) {
	var secretList corev1.SecretList
	err := h.List(ctx, &secretList, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return nil, err
	}

	credentials := []types.Credential{}

	for _, secret := range secretList.Items {
		if secret.Type != dockyardsv1.SecretTypeCredential {
			continue
		}

		credential := types.Credential{
			ID:        string(secret.UID),
			Name:      strings.TrimPrefix(secret.Name, "credential-"),
			CreatedAt: &secret.CreationTimestamp.Time,
		}

		credentialTemplateName, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
		if has {
			credential.CredentialTemplateName = &credentialTemplateName
		}

		if !secret.DeletionTimestamp.IsZero() {
			credential.DeletedAt = &secret.DeletionTimestamp.Time
		}

		credentials = append(credentials, credential)
	}

	return &credentials, nil
}

func (h *handler) CreateOrganizationCredential(ctx context.Context, organization *dockyardsv1.Organization, request *types.CredentialOptions) (*types.Credential, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-" + *request.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
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

	if request.CredentialTemplateName != nil {
		secret.Labels = map[string]string{
			dockyardsv1.LabelCredentialTemplateName: *request.CredentialTemplateName,
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

func (h *handler) DeleteOrganizationCredential(ctx context.Context, organization *dockyardsv1.Organization, credentialName string) error {
	objectKey := client.ObjectKey{
		Name:      "credential-" + credentialName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	var secret corev1.Secret
	err := h.Get(ctx, objectKey, &secret)
	if err != nil {
		return err
	}

	if secret.Type != dockyardsv1.SecretTypeCredential {
		return apierrors.NewUnauthorized("unexpected secret type")
	}

	err = h.Delete(ctx, &secret)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) GetOrganizationCredential(ctx context.Context, organization *dockyardsv1.Organization, credentialName string) (*types.Credential, error) {
	objectKey := client.ObjectKey{
		Name:      "credential-" + credentialName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	var secret corev1.Secret
	err := h.Get(ctx, objectKey, &secret)
	if err != nil {
		return nil, err
	}

	if secret.Type != dockyardsv1.SecretTypeCredential {
		return nil, apierrors.NewUnauthorized("unexpected secret type")
	}

	plaintextKeys := make(map[string]bool)

	v1Credential := types.Credential{
		ID:        string(secret.UID),
		Name:      strings.TrimPrefix(secret.Name, "credential-"),
		CreatedAt: &secret.CreationTimestamp.Time,
	}

	credentialTemplateName, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
	if has {
		var credentialTemplate dockyardsv1.CredentialTemplate
		err := h.Get(ctx, client.ObjectKey{Name: credentialTemplateName, Namespace: h.namespace}, &credentialTemplate)
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		if !apierrors.IsNotFound(err) {
			for _, option := range credentialTemplate.Spec.Options {
				if option.Plaintext {
					plaintextKeys[option.Key] = true
				}
			}
		}

		v1Credential.CredentialTemplateName = &credentialTemplateName
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

	return &v1Credential, nil
}

func (h *handler) UpdateOrganizationCredential(ctx context.Context, organization *dockyardsv1.Organization, credentialName string, request *types.CredentialOptions) error {
	objectKey := client.ObjectKey{
		Name:      "credential-" + credentialName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	var secret corev1.Secret
	err := h.Get(ctx, objectKey, &secret)
	if err != nil {
		return err
	}

	if request.Data == nil {
		return nil
	}

	patch := client.MergeFrom(secret.DeepCopy())

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	for key, value := range *request.Data {
		secret.Data[key] = value
	}

	err = h.Patch(ctx, &secret, patch)
	if err != nil {
		return err
	}

	return nil
}
