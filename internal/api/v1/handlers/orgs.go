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
	"errors"
	"strings"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=create;get;list;watch;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=members,verbs=create;get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=create;delete;get;list;watch

func (h *handler) ListGlobalOrganizations(ctx context.Context) (*[]types.Organization, error) {
	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Debug("error fetching user from context", "err", err)

		return nil, err
	}

	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelUserName: subject,
	}

	var memberList dockyardsv1.MemberList
	err = h.List(ctx, &memberList, matchingLabels)
	if err != nil {
		return nil, err
	}

	organizations := []types.Organization{}

	for _, member := range memberList.Items {
		organizationName, hasLabel := member.Labels[dockyardsv1.LabelOrganizationName]
		if !hasLabel {
			continue
		}

		key := client.ObjectKey{
			Name: organizationName,
		}

		var organization dockyardsv1.Organization
		err := h.Get(ctx, key, &organization)
		if err != nil {
			return nil, err
		}

		v1Organization := types.Organization{
			CreatedAt: organization.CreationTimestamp.Time,
			ID:        string(organization.UID),
			Name:      organization.Name,
		}

		if organization.Spec.DisplayName != "" {
			v1Organization.DisplayName = &organization.Spec.DisplayName
		}

		if !organization.DeletionTimestamp.IsZero() {
			v1Organization.DeletedAt = &organization.DeletionTimestamp.Time
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

	var user dockyardsv1.User
	err = h.Get(ctx, client.ObjectKey{Name: subject}, &user)
	if err != nil {
		return nil, err
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockyards-",
		},
	}

	err = h.Create(ctx, &namespace)
	if err != nil {
		return nil, err
	}

	member := dockyardsv1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				dockyardsv1.LabelRoleName: dockyardsv1.RoleSuperUser,
			},
			Name:      user.Name,
			Namespace: namespace.Name,
		},
		Spec: dockyardsv1.MemberSpec{
			Role: dockyardsv1.RoleSuperUser,
			UserRef: corev1.TypedLocalObjectReference{
				APIGroup: &dockyardsv1.GroupVersion.Group,
				Kind:     dockyardsv1.UserKind,
				Name:     user.Name,
			},
		},
	}

	err = h.Create(ctx, &member)
	if err != nil {
		return nil, err
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace.Name,
			Labels: map[string]string{
				dockyardsv1.LabelOrganizationName: namespace.Name,
			},
		},
		Spec: dockyardsv1.OrganizationSpec{
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
		credentialReferenceName := strings.TrimPrefix(organization.Spec.CredentialRef.Name, "credential-")
		response.CredentialReferenceName = &credentialReferenceName
	}

	return &response, nil
}

func (h *handler) UpdateGlobalOrganization(ctx context.Context, organizationName string, request *types.OrganizationOptions) error {
	objectKey := client.ObjectKey{
		Name: organizationName,
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, objectKey, &organization)
	if err != nil {
		return err
	}

	if organization.Spec.NamespaceRef == nil {
		return errors.New("qwfp")
	}

	patch := client.MergeFrom(organization.DeepCopy())
	needsPatch := false

	if request.DisplayName != nil {
		organization.Spec.DisplayName = *request.DisplayName
		needsPatch = true
	}

	if request.CredentialReferenceName != nil {
		organization.Spec.CredentialRef = &corev1.TypedObjectReference{
			Kind:      "Secret",
			Name:      "credential-" + *request.CredentialReferenceName,
			Namespace: &organization.Spec.NamespaceRef.Name,
		}

		needsPatch = true
	}

	if request.Duration != nil {
		duration, err := time.ParseDuration(*request.Duration)
		if err != nil {
			return err
		}

		organization.Spec.Duration = &metav1.Duration{
			Duration: duration,
		}

		needsPatch = true
	}

	if !needsPatch {
		return nil
	}

	err = h.Patch(ctx, &organization, patch)
	if err != nil {
		return err
	}

	return nil
}
