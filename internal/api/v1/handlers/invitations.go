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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) CreateOrganizationInvitation(ctx context.Context, organization *dockyardsv1.Organization, request *types.InvitationOptions) (*types.Invitation, error) {
	invitation := dockyardsv1.Invitation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pending-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
		Spec: dockyardsv1.InvitationSpec{
			Email: request.Email,
			Role:  dockyardsv1.OrganizationMemberRole(request.Role),
		},
	}

	if request.Duration != nil {
		duration, err := time.ParseDuration(*request.Duration)
		if err != nil {
			return nil, err
		}

		invitation.Spec.Duration = &metav1.Duration{
			Duration: duration,
		}
	}

	err := h.Create(ctx, &invitation)
	if err != nil {
		return nil, err
	}

	response := types.Invitation{
		ID:        string(invitation.UID),
		Name:      invitation.Name,
		CreatedAt: invitation.CreationTimestamp.Time,
	}

	return &response, nil
}

func (h *handler) DeleteOrganizationInvitation(ctx context.Context, organization *dockyardsv1.Organization, invitationName string) error {
	objectKey := client.ObjectKey{
		Name:      invitationName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	var invitation dockyardsv1.Invitation
	err := h.Get(ctx, objectKey, &invitation)
	if err != nil {
		return err
	}

	err = h.Delete(ctx, &invitation)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) ListOrganizationInvitations(ctx context.Context, organization *dockyardsv1.Organization) (*[]types.Invitation, error) {
	var invitationList dockyardsv1.InvitationList
	err := h.List(ctx, &invitationList, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return nil, err
	}

	result := make([]types.Invitation, len(invitationList.Items))

	for i, item := range invitationList.Items {
		result[i] = types.Invitation{
			CreatedAt: item.CreationTimestamp.Time,
			ID:        string(item.UID),
			Name:      item.Name,
			Role:      string(item.Spec.Role),
		}

		if item.Spec.Duration != nil {
			result[i].Duration = ptr.To(item.Spec.Duration.String())
			result[i].ExpiresAt = &item.GetExpiration().Time
		}
	}

	return &result, nil
}

func (h *handler) ListGlobalInvitations(ctx context.Context) (*[]types.Invitation, error) {
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
		statusError := apierrors.NewUnauthorized("unexpected users count")

		return nil, statusError
	}

	user := userList.Items[0]

	matchingFields = client.MatchingFields{
		index.EmailField: user.Spec.Email,
	}

	var invitationList dockyardsv1.InvitationList
	err = h.List(ctx, &invitationList, matchingFields)
	if err != nil {
		return nil, err
	}

	response := []types.Invitation{}

	for _, item := range invitationList.Items {
		if apiutil.HasExpired(&item) {
			continue
		}

		organization, err := apiutil.GetOwnerOrganization(ctx, h, &item)
		if err != nil {
			return nil, err
		}

		if organization == nil {
			continue
		}

		invitation := types.Invitation{
			CreatedAt:        item.CreationTimestamp.Time,
			ID:               string(item.UID),
			Name:             item.Name,
			OrganizationName: &organization.Name,
			Role:             string(item.Spec.Role),
		}

		if len(organization.Spec.DisplayName) != 0 {
			invitation.OrganizationDisplayName = &organization.Spec.DisplayName
		}

		response = append(response, invitation)
	}

	return &response, nil
}

func (h *handler) DeleteGlobalInvitation(ctx context.Context, invitationName string) error {
	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: invitationName}, &organization)
	if err != nil {
		return err
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return err
	}

	matchingFields := client.MatchingFields{
		index.UIDField: subject,
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		return err
	}

	if len(userList.Items) != 1 {
		statusError := apierrors.NewUnauthorized("unexpected users count")

		return statusError
	}

	user := userList.Items[0]

	matchingFields = client.MatchingFields{
		index.EmailField: user.Spec.Email,
	}

	var invitationList dockyardsv1.InvitationList
	err = h.List(ctx, &invitationList, matchingFields, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return err
	}

	if len(invitationList.Items) != 1 {
		statusError := apierrors.NewUnauthorized("unexpected invitations count")

		return statusError
	}

	invitation := invitationList.Items[0]

	err = h.Delete(ctx, &invitation)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) UpdateGlobalInvitation(ctx context.Context, organizationName string, _ *types.InvitationOptions) error {
	logger := middleware.LoggerFrom(ctx)

	objectKey := client.ObjectKey{
		Name: organizationName,
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, objectKey, &organization)
	if err != nil {
		return err
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return err
	}

	matchingFields := client.MatchingFields{
		index.UIDField: subject,
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		return err
	}

	if len(userList.Items) != 1 {
		statusError := apierrors.NewUnauthorized("unexpected users count")

		return statusError
	}

	user := userList.Items[0]

	matchingFields = client.MatchingFields{
		index.EmailField: user.Spec.Email,
	}

	var invitationList dockyardsv1.InvitationList
	err = h.List(ctx, &invitationList, matchingFields, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return err
	}

	if len(invitationList.Items) != 1 {
		statusError := apierrors.NewUnauthorized("unexpected invitations count")

		return statusError
	}

	invitation := invitationList.Items[0]

	patch := client.MergeFrom(organization.DeepCopy())

	memberRef := dockyardsv1.OrganizationMemberReference{
		TypedLocalObjectReference: corev1.TypedLocalObjectReference{
			APIGroup: &dockyardsv1.GroupVersion.Group,
			Kind:     dockyardsv1.UserKind,
			Name:     user.Name,
		},
		UID:  user.UID,
		Role: invitation.Spec.Role,
	}

	organization.Spec.MemberRefs = append(organization.Spec.MemberRefs, memberRef)

	err = h.Patch(ctx, &organization, patch)
	if err != nil {
		logger.Error("error patching organization", "err", err)

		return err
	}

	err = h.Delete(ctx, &invitation)
	if err != nil {
		logger.Error("error deleting invitation", "err", err)

		return err
	}

	return nil
}
