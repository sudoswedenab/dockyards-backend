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

	"github.com/sudoswedenab/dockyards-backend/templates"
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
	DockyardsExternalURL string
}

// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=verificationrequests,verbs=get;list;watch;create;delete;patch
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)

	var user dockyardsv1.User

	err := r.Get(ctx, req.NamespacedName, &user)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	verificationRequest := dockyardsv1.VerificationRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sign-up-" + user.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       user.Kind,
					Name:       user.Name,
					APIVersion: user.APIVersion,
					UID:        user.UID,
				},
			},
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &verificationRequest, func() error {
		verificationRequest.Spec.Subject = "Email Verification"
		verificationRequest.Spec.UserRef = corev1.TypedLocalObjectReference{
			Kind:     user.Kind,
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

		verificationEmail, err := renderVerificationEmail(VerificationEmailSpec{VerificationURL: r.DockyardsExternalURL + "/verify/" + code, Name: name})
		if err != nil {
			return err
		}

		verificationRequest.Spec.BodyHTML = verificationEmail.HTML
		verificationRequest.Spec.BodyText = verificationEmail.Text

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled verificationrequest", "verificationRequestName", verificationRequest.Name, "result", operationResult)
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) getNonReadyUsers(_ context.Context, obj client.Object) []ctrl.Request {
	user := obj.(*dockyardsv1.User)

	for _, c := range user.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return nil
		}
	}

	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Name: user.Name,
			},
		},
	}
}

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&dockyardsv1.VerificationRequest{}).
		Watches(
			&dockyardsv1.User{},
			handler.EnqueueRequestsFromMapFunc(r.getNonReadyUsers),
		).
		Complete(r)
}

type VerificationEmail struct {
	HTML string
	Text string
}

type VerificationEmailSpec struct {
	VerificationURL string
	Name            string
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
