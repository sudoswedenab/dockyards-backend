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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/fluxcd/pkg/runtime/patch"
	"github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/bubblebabble"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type UserReconciler struct {
	client.Client
	Config *config.ConfigManager
}

// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=users/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=verificationrequests,verbs=get;list;watch;create;delete;patch
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	var user dockyardsv1.User
	err := r.Get(ctx, req.NamespacedName, &user)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If user was not provided by dockyards, we do not care about it.
	// The system that created it is responsible for setting its ready condition to true.
	if !strings.HasPrefix(dockyardsv1.ProviderPrefixDockyards, user.Spec.ProviderID) {
		return ctrl.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(&user, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	readyCondition := conditions.Get(&user, dockyardsv1.ReadyCondition)

	// if user does not have a ready condition, set it to False
	if readyCondition == nil {
		readyCondition = &metav1.Condition{
			Type:               dockyardsv1.ReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             dockyardsv1.VerificationReasonNotVerified,
			LastTransitionTime: metav1.Now(),
		}

		conditions.Set(&user, readyCondition)
		logger.Info("user did not have ready condition, added it", "userName", user.Name, "condition", dockyardsv1.ReadyCondition, "status", metav1.ConditionFalse)

		err := patchHelper.Patch(ctx, &user)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if readyCondition.Status == metav1.ConditionTrue {
		err := authorization.ReconcileUserAuthorization(ctx, r, &user)
		if err != nil {
			conditions.MarkFalse(&user, dockyardsv1.UserAuthorizationReadyCondition, dockyardsv1.UserAuthorizationInternalErrorReason, "%s", err)

			return ctrl.Result{}, err
		}
	}

	// 1. If user was not provided by dockyards and
	if !strings.HasPrefix(user.Spec.ProviderID, dockyardsv1.ProviderPrefixDockyards) {
		return ctrl.Result{}, nil
	}

	// 2. It is not ready
	if readyCondition.Status == metav1.ConditionTrue {
		return ctrl.Result{}, nil
	}

	// 3. Then try to create/reconcile an email verification for the user.
	verificationRequest := dockyardsv1.VerificationRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sign-up-" + user.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.UserKind,
					Name:       user.Name,
					UID:        user.UID,
				},
			},
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &verificationRequest, func() error {
		verificationRequest.Spec.Subject = "Email Verification"
		verificationRequest.Spec.UserRef = corev1.TypedLocalObjectReference{
			Kind:     dockyardsv1.UserKind,
			Name:     user.Name,
			APIGroup: &dockyardsv1.GroupVersion.Group,
		}
		verificationRequest.Spec.Duration = &metav1.Duration{Duration: 30 * time.Minute}

		if verificationRequest.Spec.Code == "" {
			code, err := bubblebabble.RandomWithEntropyOfAtLeast(32)
			if err != nil {
				return err
			}
			verificationRequest.Spec.Code = code
		}

		verificationRequest.Spec.BodyText = fmt.Sprintf("Here is your email verification code: %s\nIf you have not created a dockyards account, you can ignore this email.", verificationRequest.Spec.Code)

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled verificationrequest", "verificationRequestName", verificationRequest.Name, "operationResult", operationResult)
	}

	// Verification has not yet been confirmed
	verifiedCondition := conditions.Get(&verificationRequest, dockyardsv1.VerifiedCondition)
	if verifiedCondition == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	// Verification has not yet been confirmed
	if verifiedCondition.Status != metav1.ConditionTrue {
		return ctrl.Result{Requeue: true}, nil
	}

	// If the verification has been confirmed, copy it to the user
	conditions.Set(&user, verifiedCondition)
	logger.Info("reconciled user", "userName", user.Name, "condition", dockyardsv1.ReadyCondition, "status", metav1.ConditionTrue)

	err = r.Delete(ctx, &verificationRequest)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) verificationReqeuestsToUsers(_ context.Context, obj client.Object) []ctrl.Request {
	vr := obj.(*dockyardsv1.VerificationRequest)

	if vr.Spec.UserRef.Kind != dockyardsv1.UserKind || vr.Spec.UserRef.Name == "" {
		return []ctrl.Request{}
	}

	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Name: vr.Spec.UserRef.Name,
			},
		},
	}
}

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&dockyardsv1.User{}).
		Watches(
			&dockyardsv1.VerificationRequest{},
			handler.EnqueueRequestsFromMapFunc(r.verificationReqeuestsToUsers),
		).
		Complete(r)
}
