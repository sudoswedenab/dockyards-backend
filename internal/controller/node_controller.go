package controller

import (
	"context"
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeReconciler struct {
	client.Client
	Logger         *slog.Logger
	ClusterService clusterservices.ClusterService
}

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With("name", req.Name, "namespace", req.Namespace)

	var node v1alpha1.Node
	err := r.Get(ctx, req.NamespacedName, &node)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting node", "err", err)

		return ctrl.Result{}, err
	}

	if !node.DeletionTimestamp.IsZero() {
		logger.Debug("ignoring deleted node")

		return ctrl.Result{}, nil
	}

	if node.Status.ClusterServiceID == "" {
		logger.Debug("ignoring node without cluster service id")

		return ctrl.Result{}, nil
	}

	nodeStatus, err := r.ClusterService.GetNode(node.Status.ClusterServiceID)
	if err != nil {
		logger.Error("error getting node status from cluster service", "err", err)

		return ctrl.Result{}, err
	}

	condition := meta.FindStatusCondition(nodeStatus.Conditions, v1alpha1.ReadyCondition)
	if condition == nil {
		logger.Debug("cluster service has no ready condition for node")

		return requeue, nil
	}

	patch := client.MergeFrom(node.DeepCopy())

	meta.SetStatusCondition(&node.Status.Conditions, *condition)

	err = r.Status().Patch(ctx, &node, patch)
	if err != nil {
		logger.Error("error patching node status", "err", err)

		return ctrl.Result{}, err
	}

	return requeue, nil
}

func (r *NodeReconciler) SetupWithManager(manager ctrl.Manager) error {
	scheme := manager.GetScheme()
	v1alpha1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.Node{}).Complete(r)
	if err != nil {
		return err
	}

	return nil
}
