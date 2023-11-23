package controller

import (
	"context"
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
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
	NodePoolFinalizer = "dockyards.io/backend-controller"
)

type NodePoolReconciler struct {
	client.Client
	Logger         *slog.Logger
	ClusterService clusterservices.ClusterService
}

func (r *NodePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With("name", req.Name, "namespace", req.Namespace)

	var nodePool v1alpha1.NodePool
	err := r.Get(ctx, req.NamespacedName, &nodePool)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting node pool", "err", err)

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cluster, err := apiutil.GetOwnerCluster(ctx, r.Client, &nodePool)
	if err != nil {
		logger.Error("error getting owner cluster", "err", err)

		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Debug("ignoring node pool without cluster owner")

		return ctrl.Result{}, nil
	}

	if cluster.Status.ClusterServiceID == "" {
		logger.Debug("owner cluster has no cluster service id")

		return ctrl.Result{}, nil
	}

	organization, err := apiutil.GetOwnerOrganization(ctx, r.Client, cluster)
	if err != nil {
		logger.Error("error getting owner organization", "err", err)

		return ctrl.Result{}, err
	}

	if organization == nil {
		logger.Debug("ignoring node pool without organization owner")

		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&nodePool, NodePoolFinalizer) {
		patch := client.MergeFrom(nodePool.DeepCopy())

		controllerutil.AddFinalizer(&nodePool, NodePoolFinalizer)

		err := r.Patch(ctx, &nodePool, patch)
		if err != nil {
			logger.Error("error patching node pool", "err", err)

			return ctrl.Result{}, err
		}
	}

	if !nodePool.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &nodePool, organization)
	}

	if nodePool.Status.ClusterServiceID == "" {
		logger.Debug("node pool has empty cluster service id")

		nodePoolStatus, err := r.ClusterService.CreateNodePool(organization, cluster, &nodePool)
		if err != nil {
			logger.Error("error creating node pool in cluster service", "err", err)

			readyCondition := metav1.Condition{
				Type:               v1alpha1.ReadyCondition,
				ObservedGeneration: nodePool.Generation,
				Status:             metav1.ConditionFalse,
				Reason:             v1alpha1.NodePoolReadyReason,
				Message:            err.Error(),
			}

			if !meta.IsStatusConditionPresentAndEqual(nodePool.Status.Conditions, v1alpha1.ReadyCondition, metav1.ConditionFalse) {
				logger.Debug("node pool needs status condition update")

				patch := client.MergeFrom(nodePool.DeepCopy())

				meta.SetStatusCondition(&nodePool.Status.Conditions, readyCondition)

				err := r.Status().Patch(ctx, &nodePool, patch)
				if err != nil {
					logger.Error("error patching node pool status", "err", err)

					return ctrl.Result{}, err
				}
			}
		}

		if nodePoolStatus != nil {
			patch := client.MergeFrom(nodePool.DeepCopy())

			nodePool.Status.ClusterServiceID = nodePoolStatus.ClusterServiceID

			err = r.Status().Patch(ctx, &nodePool, patch)
			if err != nil {
				logger.Error("error patching node pool status", "err", err)

				return ctrl.Result{}, err
			}
		}

		return requeue, nil
	}

	nodePoolStatus, err := r.ClusterService.GetNodePool(nodePool.Status.ClusterServiceID)
	if err != nil {
		logger.Error("error geting node pool from cluster service", "err", err)

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

	err = r.Status().Patch(ctx, &nodePool, patch)
	if err != nil {
		logger.Error("error patching status conditions", "err", err)

		return ctrl.Result{}, err
	}

	nodeList, err := r.ClusterService.GetNodes(&nodePool)
	for _, nodeItem := range nodeList.Items {
		objectKey := client.ObjectKey{
			Name:      nodeItem.Name,
			Namespace: nodePool.Namespace,
		}

		var node v1alpha1.Node
		err := r.Get(ctx, objectKey, &node)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting node", "err", err)

			return ctrl.Result{}, err
		}

		if apierrors.IsNotFound(err) {
			logger.Debug("node not found")

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

			err := r.Create(ctx, &node)
			if err != nil {
				logger.Error("error creating node", "err", err)

				return ctrl.Result{}, err
			}

			patch := client.MergeFrom(node.DeepCopy())

			node.Status = v1alpha1.NodeStatus{
				ClusterServiceID: nodeItem.Status.ClusterServiceID,
			}

			err = r.Status().Patch(ctx, &node, patch)
			if err != nil {
				logger.Error("error patching node", "err", err)

				return ctrl.Result{}, err
			}
		}
	}

	return requeue, nil
}

func (r *NodePoolReconciler) reconcileDelete(ctx context.Context, nodePool *v1alpha1.NodePool, organization *v1alpha2.Organization) (ctrl.Result, error) {
	logger := r.Logger.With("name", nodePool.Name, "namespace", nodePool.Namespace)

	if nodePool.Status.ClusterServiceID != "" {
		err := r.ClusterService.DeleteNodePool(organization, nodePool.Status.ClusterServiceID)
		if err != nil {
			logger.Error("error deleting node pool from cluster service", "err", err)

			return ctrl.Result{}, err
		}
	}

	patch := client.MergeFrom(nodePool.DeepCopy())

	controllerutil.RemoveFinalizer(nodePool, NodePoolFinalizer)

	err := r.Patch(ctx, nodePool, patch)
	if err != nil {
		logger.Error("error removing finalizer from node pool", "err", err)

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NodePoolReconciler) SetupWithManager(manager ctrl.Manager) error {
	scheme := manager.GetScheme()
	v1alpha1.AddToScheme(scheme)
	v1alpha2.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.NodePool{}).Complete(r)
	if err != nil {
		return err
	}

	return nil
}
