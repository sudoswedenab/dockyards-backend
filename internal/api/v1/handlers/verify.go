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

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) UpdateGlobalVerificationRequest(ctx context.Context, request *types.VerifyOptions) error {
	if request.Type != dockyardsv1.RequestTypeAccount {
		return apierrors.NewUnauthorized("invalid verification type")
	}

	var verificationRequestList dockyardsv1.VerificationRequestList
	err := h.List(ctx, &verificationRequestList, client.MatchingFields{
		index.CodeField: request.Code,
	})
	if err != nil {
		return err
	}
	if len(verificationRequestList.Items) == 0 {
		return apierrors.NewUnauthorized("could not find verification request")
	}
	if len(verificationRequestList.Items) > 1 {
		// Verification requests should be reasonably unique, but could still collide.
		return apierrors.NewUnauthorized("unexpected multiple verification requests with the same code")
	}
	verificationRequest := verificationRequestList.Items[0]

	// FIXME: The verification request should be cleaned up with the user account.
	expiresAt := verificationRequest.GetExpiration()
	if expiresAt == nil {
		return apierrors.NewUnauthorized("account verification request needs an expiration")
	}
	if time.Now().After(expiresAt.Time) {
		return apierrors.NewUnauthorized("verification request has expired")
	}

	patch := client.MergeFrom(verificationRequest.DeepCopy())

	condition := metav1.Condition{
		Type:    dockyardsv1.VerifiedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  dockyardsv1.VerificationReasonVerified,
		Message: "Verified by code",
	}

	meta.SetStatusCondition(&verificationRequest.Status.Conditions, condition)

	err = h.Client.Status().Patch(ctx, &verificationRequest, patch)
	if err != nil {
		return err
	}

	return nil
}
