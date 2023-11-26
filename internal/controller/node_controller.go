package controller

import (
	"context"
	"log/slog"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=nodes,verbs=get;delete;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=nodes/status,verbs=patch

var (
	NodeRequeue = ctrl.Result{Requeue: true, RequeueAfter: time.Duration(time.Second * 30)}
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

	if nodeStatus == nil {
		logger.Info("deleting node missing in cluster service")

		err := r.Delete(ctx, &node)
		if err != nil {
			logger.Error("error deleting node")

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	needsPatch := false

	patch := client.MergeFrom(node.DeepCopy())

	provisionedCondition := meta.FindStatusCondition(nodeStatus.Conditions, v1alpha1.ProvisionedCondition)
	if provisionedCondition != nil {
		if !meta.IsStatusConditionPresentAndEqual(node.Status.Conditions, provisionedCondition.Type, provisionedCondition.Status) {
			meta.SetStatusCondition(&node.Status.Conditions, *provisionedCondition)

			needsPatch = true
		}
	}

	readyCondition := meta.FindStatusCondition(nodeStatus.Conditions, v1alpha1.ReadyCondition)
	if readyCondition != nil {
		if !meta.IsStatusConditionPresentAndEqual(node.Status.Conditions, readyCondition.Type, readyCondition.Status) {
			meta.SetStatusCondition(&node.Status.Conditions, *readyCondition)

			needsPatch = true
		}
	}

	if readyCondition == nil && provisionedCondition != nil {
		condition := metav1.Condition{
			Type:   v1alpha1.ReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: v1alpha1.ProvisioningFailedReason,
		}

		meta.SetStatusCondition(&node.Status.Conditions, condition)

		needsPatch = true
	}

	if node.Status.CloudServiceID != nodeStatus.CloudServiceID {
		node.Status.CloudServiceID = nodeStatus.CloudServiceID

		needsPatch = true
	}

	if needsPatch {
		logger.Debug("node status needs patch")

		err = r.Status().Patch(ctx, &node, patch)
		if err != nil {
			logger.Error("error patching node status", "err", err)

			return ctrl.Result{}, err
		}
	}

	return NodeRequeue, nil
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
