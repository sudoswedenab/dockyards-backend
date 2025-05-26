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

	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3"
	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/fluxcd/pkg/runtime/patch"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=workloads,verbs=get;list;patch;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=workloadinventories,verbs=get;list;watch

type WorkloadReconciler struct {
	client.Client
}

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	var workload dockyardsv1.Workload
	err := r.Get(ctx, req.NamespacedName, &workload)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	patchHelper, err := patch.NewHelper(&workload, r)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := patchHelper.Patch(ctx, &workload)
		if err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelWorkloadName: workload.Name,
	}

	var workloadInventoryList dockyardsv1.WorkloadInventoryList
	err = r.List(ctx, &workloadInventoryList, matchingLabels, client.InNamespace(workload.Namespace))
	if err != nil {
		return ctrl.Result{}, err
	}

	urls := []string{}

	for _, workloadInventory := range workloadInventoryList.Items {
		urls = append(urls, workloadInventory.Spec.URLs...)
	}

	workload.Status.URLs = urls

	conditions.MarkTrue(&workload, dockyardsv1.WorkloadInventoryReadyCondition, dockyardsv1.ReadyReason, "")

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) workloadInventoryToWorkload(_ context.Context, obj client.Object) []ctrl.Request {
	labels := obj.GetLabels()

	workloadName, hasLabel := labels[dockyardsv1.LabelWorkloadName]
	if !hasLabel {
		return nil
	}

	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      workloadName,
				Namespace: obj.GetNamespace(),
			},
		},
	}
}

func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(mgr).
		For(&dockyardsv1.Workload{}).
		Watches(
			&dockyardsv1.WorkloadInventory{},
			handler.EnqueueRequestsFromMapFunc(r.workloadInventoryToWorkload),
		).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
