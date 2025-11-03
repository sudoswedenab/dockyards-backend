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

	"github.com/google/uuid"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"

	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/fluxcd/pkg/runtime/patch"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	"github.com/sudoswedenab/dockyards-backend/templates"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type UserReconciler struct {
	client.Client
	DockyardsConfig *dyconfig.DockyardsConfig
}

type VerificationEmail struct {
	HTML string
	Text string
}

type VerificationEmailSpec struct {
	VerificationURL string
	Name            string
}

// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=users/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=verificationrequests,verbs=get;list;watch;create;delete;patch
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)
	var user dockyardsv1.User
	err := r.Get(ctx, req.NamespacedName, &user)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	patchHelper, err := patch.NewHelper(&user, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := patchHelper.Patch(ctx, &user)
		if err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	readyCondition := conditions.Get(&user, dockyardsv1.ReadyCondition)

	// if User doesn't have a ready condition, set it to False
	if readyCondition == nil {
		condition := metav1.Condition{
			Type:               dockyardsv1.ReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             dockyardsv1.VerificationReasonNotVerified,
			Message:            "",
			LastTransitionTime: metav1.Now(),
		}

		conditions.Set(&user, &condition)
		logger.Info("reconciled user", "userName", user.Name, "condition", dockyardsv1.ReadyCondition, "status", metav1.ConditionFalse)

		return ctrl.Result{Requeue: true}, nil
	}

	// if User is not ready, ensure that it has a corresponding VerificationRequest with a Verified condition
	if readyCondition.Status == metav1.ConditionFalse {
		verificationRequest, operationResult, err := r.reconcileVerificationRequest(ctx, &user)
		if err != nil {
			return ctrl.Result{}, err
		}

		if operationResult != controllerutil.OperationResultNone {
			logger.Info("reconciled verificationrequest", "verificationRequestName", verificationRequest.Name, "result", operationResult)
		}

		if operationResult == controllerutil.OperationResultCreated {
			return ctrl.Result{Requeue: true}, nil
		}

		verifiedCondition := conditions.Get(verificationRequest, dockyardsv1.VerifiedCondition)

		// if VerificationRequest has Verified set to True, mark User as Ready
		if verifiedCondition != nil && verifiedCondition.Status == metav1.ConditionTrue {
			condition := metav1.Condition{
				Type:               dockyardsv1.ReadyCondition,
				Status:             verifiedCondition.Status,
				Reason:             verifiedCondition.Reason,
				Message:            verifiedCondition.Message,
				LastTransitionTime: verifiedCondition.LastTransitionTime,
			}
			conditions.Set(&user, &condition)
			logger.Info("reconciled user", "userName", user.Name, "condition", dockyardsv1.ReadyCondition, "status", metav1.ConditionTrue)

			return ctrl.Result{Requeue: true}, nil
		}
	}

	// if User is Ready, make sure VerificationRequest is deleted
	if readyCondition.Status == metav1.ConditionTrue {
		vr := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{Name: "sign-up-" + user.Name},
		}

		err := r.Get(ctx, client.ObjectKeyFromObject(&vr), &vr)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		err = r.Delete(ctx, &vr)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		logger.Info("reconciled verificationrequest", "verificationRequestName", "sign-up-"+user.Name, "result", "deleted")
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) reconcileVerificationRequest(ctx context.Context, user *dockyardsv1.User) (*dockyardsv1.VerificationRequest, controllerutil.OperationResult, error) {
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

		code := verificationRequest.Spec.Code
		if code == "" {
			randomUUID, err := uuid.NewRandom()
			if err != nil {
				return err
			}

			code = randomUUID.String()
			verificationRequest.Spec.Code = code
		}

		name := user.Name
		if user.Spec.DisplayName != "" {
			name = user.Spec.DisplayName
		}

		externalURL := r.DockyardsConfig.GetConfigKey(dyconfig.KeyExternalURL, "http://localhost:9000")
		verificationEmail, err := renderVerificationEmail(VerificationEmailSpec{VerificationURL: externalURL + "/verify/" + code, Name: name})
		if err != nil {
			return err
		}

		verificationRequest.Spec.BodyHTML = verificationEmail.HTML
		verificationRequest.Spec.BodyText = verificationEmail.Text

		return nil
	})
	if err != nil {
		return nil, controllerutil.OperationResultNone, err
	}

	return &verificationRequest, operationResult, nil
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

func renderVerificationEmail(spec VerificationEmailSpec) (VerificationEmail, error) {
	html, err := renderFromTemplate(spec, "sign-up-confirmation.html.tmpl")
	if err != nil {
		return VerificationEmail{}, err
	}

	text, err := renderFromTemplate(spec, "sign-up-confirmation.txt.tmpl")
	if err != nil {
		return VerificationEmail{}, err
	}

	return VerificationEmail{
		HTML: html,
		Text: text,
	}, nil
}

func renderFromTemplate(spec VerificationEmailSpec, template string) (string, error) {
	body := templates.Get(template)
	if body == nil {
		return "", fmt.Errorf("unable to find template %s", template)
	}

	var builder strings.Builder
	err := body.Execute(&builder, spec)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
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
