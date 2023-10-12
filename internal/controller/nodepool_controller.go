package controller

import (
	"context"
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters;organizations,verbs=get
// +kubebuilder:rbac:groups=dockyards.io,resources=nodes,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=dockyards.io,resources=nodes/status,verbs=patch

const (
	nodePoolFinalizer = "dockyards.io/backend-controller"
)

type nodePoolController struct {
	client.Client
	logger         *slog.Logger
	clusterService clusterservices.ClusterService
}

func (c *nodePoolController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var nodePool v1alpha1.NodePool
	err := c.Get(ctx, req.NamespacedName, &nodePool)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cluster, err := c.getOwnerCluster(ctx, &nodePool)
	if err != nil {
		c.logger.Error("error getting owner cluster", "err", err)

		return ctrl.Result{}, err
	}

	if cluster == nil {
		c.logger.Debug("ignoring node pool without cluster owner")

		return requeue, nil
	}

	if cluster.Status.ClusterServiceID == "" {
		c.logger.Debug("owner cluster has no cluster service id")

		return requeue, nil
	}

	organization, err := c.getOwnerOrganization(ctx, cluster)
	if err != nil {
		c.logger.Error("error getting owner organization", "err", err)

		return ctrl.Result{}, err
	}

	if organization == nil {
		c.logger.Debug("ignoring node pool with cluster owner without organization owner")

		return requeue, nil
	}

	if !controllerutil.ContainsFinalizer(&nodePool, nodePoolFinalizer) {
		patch := client.MergeFrom(nodePool.DeepCopy())
		controllerutil.AddFinalizer(&nodePool, nodePoolFinalizer)
		err := c.Patch(ctx, &nodePool, patch)
		if err != nil {
			c.logger.Error("error adding finalizer to node pool", "err", err)

			return ctrl.Result{}, err
		}
	}

	if !nodePool.DeletionTimestamp.IsZero() {
		return c.reconcileDelete(ctx, &nodePool, organization)
	}

	if nodePool.Status.ClusterServiceID == "" {
		c.logger.Debug("node pool has empty cluster service id")

		nodePoolStatus, err := c.clusterService.CreateNodePool(organization, cluster, &nodePool)
		if err != nil {
			c.logger.Error("error creating node pool in cluster service", "err", err)

			return ctrl.Result{}, err
		}

		patch := client.MergeFrom(nodePool.DeepCopy())
		nodePool.Status = *nodePoolStatus
		err = c.Status().Patch(ctx, &nodePool, patch)
		if err != nil {
			c.logger.Error("error patching node pool status", "err", err)

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	nodePoolStatus, err := c.clusterService.GetNodePool(nodePool.Status.ClusterServiceID)
	if err != nil {
		c.logger.Error("error geting node pool from cluster service", "err", err)

		return ctrl.Result{}, err
	}

	patch := client.MergeFrom(nodePool.DeepCopy())

	condition := metav1.Condition{
		Type:    v1alpha1.ReadyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.NodePoolReadyReason,
		Message: nodePoolStatus.ClusterServiceID,
	}
	meta.SetStatusCondition(&nodePool.Status.Conditions, condition)

	err = c.Status().Patch(ctx, &nodePool, patch)
	if err != nil {
		c.logger.Error("error patching status conditions", "err", err)

		return ctrl.Result{}, err
	}

	nodeList, err := c.clusterService.GetNodes(&nodePool)
	for _, nodeItem := range nodeList.Items {
		c.logger.Debug("node item", "name", nodeItem.Name)

		objectKey := client.ObjectKey{
			Name:      nodeItem.Name,
			Namespace: nodePool.Namespace,
		}

		var node v1alpha1.Node
		err := c.Get(ctx, objectKey, &node)
		if client.IgnoreNotFound(err) != nil {
			c.logger.Error("error getting node", "err", err)

			return ctrl.Result{}, err
		}

		if apierrors.IsNotFound(err) {
			node := v1alpha1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeItem.Name,
					Namespace: nodePool.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         v1alpha1.GroupVersion.String(),
							Kind:               v1alpha1.NodePoolKind,
							Name:               nodePool.Name,
							UID:                nodePool.UID,
							BlockOwnerDeletion: util.Ptr(true),
						},
					},
				},
			}

			err := c.Create(ctx, &node)
			if err != nil {
				c.logger.Error("error creating node", "err", err)

				return ctrl.Result{}, err
			}

			patch := client.MergeFrom(node.DeepCopy())

			node.Status = v1alpha1.NodeStatus{
				ClusterServiceID: nodeItem.Status.ClusterServiceID,
			}

			err = c.Status().Patch(ctx, &node, patch)
			if err != nil {
				c.logger.Error("error patching node", "err", err)

				return ctrl.Result{}, err
			}
		}
	}

	return requeue, nil
}

func (c *nodePoolController) reconcileDelete(ctx context.Context, nodePool *v1alpha1.NodePool, organization *v1alpha1.Organization) (ctrl.Result, error) {
	c.logger.Debug("deleted node pool")

	if nodePool.Status.ClusterServiceID != "" {
		err := c.clusterService.DeleteNodePool(organization, nodePool.Status.ClusterServiceID)
		if err != nil {
			c.logger.Error("error deleting node pool from cluster service", "err", err)

			return ctrl.Result{}, err
		}

		c.logger.Debug("deleted node pool from cluster service", "id", nodePool.Status.ClusterServiceID)
	}

	patch := client.MergeFrom(nodePool.DeepCopy())
	controllerutil.RemoveFinalizer(nodePool, nodePoolFinalizer)

	err := c.Patch(ctx, nodePool, patch)
	if err != nil {
		c.logger.Error("error removing finalizer from node pool", "err", err)

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (c *nodePoolController) getOwnerCluster(ctx context.Context, nodePool *v1alpha1.NodePool) (*v1alpha1.Cluster, error) {
	for _, ownerReference := range nodePool.GetOwnerReferences() {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != v1alpha1.ClusterKind {
			continue
		}

		objectKey := client.ObjectKey{
			Name:      ownerReference.Name,
			Namespace: nodePool.Namespace,
		}

		var cluster v1alpha1.Cluster
		err := c.Get(ctx, objectKey, &cluster)
		if err != nil {
			return nil, err
		}

		return &cluster, nil
	}

	return nil, nil
}

func (c *nodePoolController) getOwnerOrganization(ctx context.Context, cluster *v1alpha1.Cluster) (*v1alpha1.Organization, error) {
	for _, ownerReference := range cluster.GetOwnerReferences() {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != v1alpha1.OrganizationKind {
			continue
		}

		objectKey := client.ObjectKey{
			Name:      ownerReference.Name,
			Namespace: cluster.Namespace,
		}

		var organization v1alpha1.Organization
		err := c.Get(ctx, objectKey, &organization)
		if err != nil {
			return nil, err
		}

		return &organization, nil
	}

	return nil, nil
}

func NewNodePoolController(manager ctrl.Manager, clusterService clusterservices.ClusterService, logger *slog.Logger) error {
	client := manager.GetClient()

	c := nodePoolController{
		Client:         client,
		clusterService: clusterService,
		logger:         logger,
	}

	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.NodePool{}).Complete(&c)
	if err != nil {
		return err
	}

	return nil
}
