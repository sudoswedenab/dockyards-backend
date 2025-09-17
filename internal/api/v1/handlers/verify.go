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

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) UpdateGlobalVerificationRequest(ctx context.Context, request *types.VerifyOptions) error {
	if request.Type != dockyardsv1.RequestTypeAccount {
		return apierrors.NewUnauthorized("invalid verification type")
	}

	var vr dockyardsv1.VerificationRequest
	var vrl dockyardsv1.VerificationRequestList

	err := h.List(ctx, &vrl)
	if err != nil {
		return err
	}

	for _, verification := range vrl.Items {
		if verification.Spec.Code == request.Code {
			vr = verification

			break
		}
	}

	if vr.Name == "" {
		return apierrors.NewUnauthorized("invalid verification code")
	}

	patch := client.MergeFrom(vr.DeepCopy())

	condition := metav1.Condition{
		Type:    dockyardsv1.VerifiedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  dockyardsv1.VerificationReasonVerified,
		Message: "Verified by link",
	}

	meta.SetStatusCondition(&vr.Status.Conditions, condition)

	err = h.Client.Status().Patch(ctx, &vr, patch)
	if err != nil {
		return err
	}

	return nil
}
