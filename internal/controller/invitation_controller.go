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

	"github.com/fluxcd/pkg/runtime/patch"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=invitations,verbs=delete;get;list;patch;watch

type InvitationReconciler struct {
	client.Client
}

func (r *InvitationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	var invitation dockyardsv1.Invitation
	err := r.Get(ctx, req.NamespacedName, &invitation)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if apiutil.HasExpired(&invitation) {
		err := r.Delete(ctx, &invitation, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	patchHelper, err := patch.NewHelper(&invitation, r)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := patchHelper.Patch(ctx, &invitation)
		if err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	expiration := invitation.GetExpiration()
	invitation.Status.ExpirationTimestamp = expiration

	if expiration != nil {
		requeueAfter := time.Until(expiration.Time)

		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *InvitationReconciler) SetupWithManger(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(mgr).For(&dockyardsv1.Invitation{}).Complete(r)
	if err != nil {
		return err
	}

	return nil
}
