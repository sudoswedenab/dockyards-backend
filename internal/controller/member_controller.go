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
	"time"

	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/fluxcd/pkg/runtime/patch"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=members/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=members,verbs=get;list;patch;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=users,verbs=get;list;watch

type MemberReconciler struct {
	client.Client
}

func (r *MemberReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	var member dockyardsv1.Member
	err := r.Get(ctx, req.NamespacedName, &member)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !member.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(&member, r)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := patchHelper.Patch(ctx, &member)
		if err != nil {
			result = ctrl.Result{}
			reterr = err
		}
	}()

	result, err = r.reconcileAuthorization(ctx, &member)
	if err != nil {
		return result, err
	}

	result, err = r.reconcileInfo(ctx, &member)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *MemberReconciler) reconcileAuthorization(ctx context.Context, member *dockyardsv1.Member) (ctrl.Result, error) {
	err := authorization.ReconcileMemberAuthorization(ctx, r, member)
	if err != nil {
		conditions.MarkFalse(member, dockyardsv1.MemberAuthorizationReadyCondition, dockyardsv1.MemberAuthorizationInternalErrorReason, "%s", err)

		return ctrl.Result{}, nil
	}

	conditions.MarkTrue(member, dockyardsv1.MemberAuthorizationReadyCondition, dockyardsv1.ReadyReason, "")

	return ctrl.Result{}, nil
}

func (r *MemberReconciler) reconcileInfo(ctx context.Context, member *dockyardsv1.Member) (ctrl.Result, error) {
	organization, err := apiutil.GetOrganizationByNamespaceRef(ctx, r, member.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if member.Labels == nil {
		member.Labels = map[string]string{}
	}
	member.Labels[dockyardsv1.LabelRoleName] = string(member.Spec.Role)
	member.Labels[dockyardsv1.LabelUserName] = member.Spec.UserRef.Name
	member.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

	key := client.ObjectKey{
		Name: member.Spec.UserRef.Name,
	}
	var user dockyardsv1.User
	err = r.Get(ctx, key, &user)
	if errors.IsNotFound(err) {
		// User has not been created yet
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if user.Spec.Email == "" {
		member.Status.Email = nil
	} else {
		member.Status.Email = &user.Spec.Email
	}

	if user.Spec.DisplayName == "" {
		member.Status.DisplayName = nil
	} else {
		member.Status.DisplayName = &user.Spec.DisplayName
	}

	return ctrl.Result{}, nil
}

func (r *MemberReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(mgr).For(&dockyardsv1.Member{}).Complete(r)
	if err != nil {
		return err
	}

	return nil
}
