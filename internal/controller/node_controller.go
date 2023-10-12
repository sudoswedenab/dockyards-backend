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

type nodeController struct {
	client.Client
	logger         *slog.Logger
	clusterService clusterservices.ClusterService
}

func (c *nodeController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var node v1alpha1.Node
	err := c.Get(ctx, req.NamespacedName, &node)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if node.Status.ClusterServiceID == "" {
		c.logger.Debug("ignoring node without cluster service id")

		return ctrl.Result{}, nil
	}

	nodeStatus, err := c.clusterService.GetNode(node.Status.ClusterServiceID)
	if err != nil {
		c.logger.Error("error getting node pool status from cluster service", "err", err)

		return ctrl.Result{}, err
	}

	condition := meta.FindStatusCondition(nodeStatus.Conditions, v1alpha1.ReadyCondition)
	if condition == nil {
		c.logger.Debug("cluster service has no ready condition for node")

		return requeue, nil
	}

	patch := client.MergeFrom(node.DeepCopy())
	meta.SetStatusCondition(&node.Status.Conditions, *condition)

	err = c.Status().Patch(ctx, &node, patch)
	if err != nil {
		c.logger.Error("error patching node status", "err", err)

		return ctrl.Result{}, err
	}

	return requeue, nil
}

func NewNodeController(manager ctrl.Manager, clusterService clusterservices.ClusterService, logger *slog.Logger) error {
	c := nodeController{
		Client:         manager.GetClient(),
		logger:         logger,
		clusterService: clusterService,
	}

	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.Node{}).Complete(&c)
	if err != nil {
		return err
	}

	return nil
}
