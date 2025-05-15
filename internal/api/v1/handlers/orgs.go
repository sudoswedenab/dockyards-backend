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
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=create;get;list;watch;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=create;delete;get;list;watch

func (h *handler) ListGlobalOrganizations(ctx context.Context) (*[]types.Organization, error) {
	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Debug("error fetching user from context", "err", err)

		return nil, err
	}

	matchingFields := client.MatchingFields{
		index.MemberReferencesField: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error listing organizations in kubernetes", "err", err)

		return nil, err
	}

	organizations := []types.Organization{}

	for _, organization := range organizationList.Items {
		v1Organization := types.Organization{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(organization.UID),
			Name:      organization.Name,
		}

		if !organization.DeletionTimestamp.IsZero() {
			v1Organization.DeletedAt = &organization.CreationTimestamp.Time
		}

		if organization.Spec.Duration != nil {
			duration := organization.Spec.Duration.String()

			v1Organization.Duration = &duration
		}

		readyCondition := meta.FindStatusCondition(organization.Status.Conditions, dockyardsv1.ReadyCondition)
		if readyCondition != nil {
			v1Organization.UpdatedAt = &readyCondition.LastTransitionTime.Time
			v1Organization.Condition = &readyCondition.Reason
		}

		organizations = append(organizations, v1Organization)
	}

	return &organizations, nil
}

func (h *handler) CreateGlobalOrganization(ctx context.Context, request *types.OrganizationOptions) (*types.Organization, error) {
	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return nil, err
	}

	matchingFields := client.MatchingFields{
		index.UIDField: subject,
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		return nil, err
	}

	if len(userList.Items) != 1 {
		statusError := apierrors.NewInternalError(nil)

		return nil, statusError
	}

	user := userList.Items[0]

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockyards-",
		},
	}

	err = h.Create(ctx, &namespace)
	if err != nil {
		return nil, err
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace.Name,
		},
		Spec: dockyardsv1.OrganizationSpec{
			MemberRefs: []dockyardsv1.OrganizationMemberReference{
				{
					TypedLocalObjectReference: corev1.TypedLocalObjectReference{
						Kind: dockyardsv1.UserKind,
						Name: user.Name,
					},
					Role: dockyardsv1.OrganizationMemberRoleSuperUser,
					UID:  user.UID,
				},
			},
			NamespaceRef: &corev1.LocalObjectReference{
				Name: namespace.Name,
			},
			ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
		},
	}

	if request.DisplayName != nil {
		organization.Spec.DisplayName = *request.DisplayName
	}

	if request.Duration != nil {
		duration, err := time.ParseDuration(*request.Duration)
		if err != nil {
			return nil, err
		}

		organization.Spec.Duration = &metav1.Duration{
			Duration: duration,
		}
	}

	if request.VoucherCode != nil {
		matchingFields := client.MatchingFields{
			index.CodeField: *request.VoucherCode,
		}

		var organizationVoucherList dockyardsv1.OrganizationVoucherList
		err := h.List(ctx, &organizationVoucherList, matchingFields, client.InNamespace(h.namespace))
		if err != nil {
			return nil, err
		}

		if len(organizationVoucherList.Items) != 1 {
			statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.OrganizationKind).GroupKind(), "", nil)

			return nil, statusError
		}

		organizationVoucher := organizationVoucherList.Items[0]

		if organizationVoucher.Status.Redeemed {
			statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.OrganizationKind).GroupKind(), "", nil)

			return nil, statusError
		}

		organization.Annotations = map[string]string{
			dockyardsv1.AnnotationVoucherCode: organizationVoucher.Spec.Code,
		}
	}

	err = h.Create(ctx, &organization)
	if err != nil {
		return nil, err
	}

	patch := client.MergeFrom(namespace.DeepCopy())

	namespace.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: dockyardsv1.GroupVersion.String(),
			Kind:       dockyardsv1.OrganizationKind,
			Name:       organization.Name,
			UID:        organization.UID,
		},
	}

	namespace.Labels = map[string]string{
		dockyardsv1.LabelOrganizationName: organization.Name,
	}

	err = h.Patch(ctx, &namespace, patch)
	if err != nil {
		return nil, err
	}

	response := types.Organization{
		CreatedAt: organization.CreationTimestamp.Time,
		ID:        string(organization.UID),
		Name:      organization.Name,
	}

	if organization.Spec.DisplayName != "" {
		response.DisplayName = &organization.Spec.DisplayName
	}

	return &response, nil
}

func (h *handler) DeleteGlobalOrganization(ctx context.Context, resourceName string) error {
	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: resourceName}, &organization)
	if err != nil {
		return err
	}

	err = h.Delete(ctx, &organization, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) GetGlobalOrganization(ctx context.Context, organizationName string) (*types.Organization, error) {
	objectKey := client.ObjectKey{
		Name: organizationName,
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, objectKey, &organization)
	if err != nil {
		return nil, err
	}

	response := types.Organization{
		CreatedAt: organization.CreationTimestamp.Time,
		ID:        string(organization.UID),
		Name:      organization.Name,
	}

	readyCondition := meta.FindStatusCondition(organization.Status.Conditions, dockyardsv1.ReadyCondition)
	if readyCondition != nil {
		response.UpdatedAt = &readyCondition.LastTransitionTime.Time
		response.Condition = &readyCondition.Reason
	}

	if len(organization.Spec.DisplayName) > 0 {
		response.DisplayName = &organization.Spec.DisplayName
	}

	if organization.Spec.ProviderID != nil {
		response.ProviderID = organization.Spec.ProviderID
	}

	voucherCode, hasAnnotation := organization.Annotations[dockyardsv1.AnnotationVoucherCode]
	if hasAnnotation {
		response.VoucherCode = &voucherCode
	}

	expiration := organization.GetExpiration()
	if expiration != nil {
		response.ExpiresAt = &expiration.Time
	}

	if !organization.DeletionTimestamp.IsZero() {
		response.DeletedAt = &organization.DeletionTimestamp.Time
	}

	if organization.Spec.CredentialRef != nil {
		response.CredentialReferenceName = &organization.Spec.CredentialRef.Name
	}

	return &response, nil
}
