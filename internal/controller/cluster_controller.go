package controller

import (
	"context"
	"log/slog"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch

var (
	requeue = ctrl.Result{Requeue: true, RequeueAfter: time.Duration(time.Minute)}
)

const (
	clusterFinalizer = "dockyards.io/backend-controller"
)

type clusterController struct {
	client.Client
	logger         *slog.Logger
	clusterService clusterservices.ClusterService
}

func (c *clusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cluster v1alpha1.Cluster
	err := c.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	organization, err := c.getOwnerOrganization(ctx, &cluster)
	if err != nil {
		c.logger.Error("error getting owner organization", "err", err)

		return ctrl.Result{}, err
	}

	if organization == nil {
		c.logger.Info("ignoring cluster without organization owner")

		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&cluster, clusterFinalizer) {
		patch := client.MergeFrom(cluster.DeepCopy())
		c.logger.Debug("finalizer patch", "patch", patch)

		controllerutil.AddFinalizer(&cluster, clusterFinalizer)
		err := c.Patch(ctx, &cluster, patch)
		if err != nil {
			c.logger.Error("error adding finalizer to cluster", "err", err)

			return ctrl.Result{}, err
		}

		c.logger.Debug("added finalizer to cluster")
	}

	if !cluster.DeletionTimestamp.IsZero() {
		return c.reconcileDelete(ctx, &cluster, organization)
	}

	if cluster.Status.ClusterServiceID == "" {
		c.logger.Debug("cluster has empty cluster service id")

		clusterOptions := v1.ClusterOptions{
			Name: cluster.Name,
		}

		createdCluster, err := c.clusterService.CreateCluster(organization, &clusterOptions)
		if err != nil {
			c.logger.Error("error creating cluster in cluster service", "err", err)

			return ctrl.Result{}, err
		}

		c.logger.Debug("created cluster in cluster service", "id", createdCluster.ID)

		patch := client.MergeFrom(cluster.DeepCopy())

		cluster.Status.ClusterServiceID = createdCluster.ID
		err = c.Status().Patch(ctx, &cluster, patch)
		if err != nil {
			c.logger.Error("error patching cluster status", "err", err)

			return ctrl.Result{}, err
		}

		return requeue, nil
	}

	existingCluster, err := c.clusterService.GetCluster(cluster.Status.ClusterServiceID)
	if err != nil {
		c.logger.Error("error getting cluster from cluster service", "err", err)

		return ctrl.Result{}, err
	}

	condition := metav1.Condition{
		Type:    v1alpha1.ReadyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ClusterReadyReason,
		Message: existingCluster.State,
	}

	patch := client.MergeFrom(cluster.DeepCopy())
	meta.SetStatusCondition(&cluster.Status.Conditions, condition)
	err = c.Status().Patch(ctx, &cluster, patch)
	if err != nil {
		c.logger.Error("error updating cluster status conditions", "err", err)

		return ctrl.Result{}, err
	}

	return requeue, nil
}

func (c *clusterController) reconcileDelete(ctx context.Context, cluster *v1alpha1.Cluster, organization *v1alpha1.Organization) (ctrl.Result, error) {
	if cluster.Status.ClusterServiceID == "" {
		c.logger.Warn("cluster has no cluster service id")
	}

	if cluster.Status.ClusterServiceID != "" {
		v1Cluster := v1.Cluster{
			ID:   cluster.Status.ClusterServiceID,
			Name: cluster.Name,
		}

		err := c.clusterService.DeleteCluster(organization, &v1Cluster)
		if err != nil {
			c.logger.Error("error deleting cluster from cluster service", "err", err)

			return ctrl.Result{}, err
		}
	}

	patch := client.MergeFrom(cluster.DeepCopy())
	controllerutil.RemoveFinalizer(cluster, clusterFinalizer)
	err := c.Patch(ctx, cluster, patch)
	if err != nil {
		c.logger.Error("error removing finalizer from cluster", "err", err)

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (c *clusterController) getOwnerOrganization(ctx context.Context, cluster *v1alpha1.Cluster) (*v1alpha1.Organization, error) {
	for _, ownerReference := range cluster.GetOwnerReferences() {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}
		if ownerReference.Kind != v1alpha1.OrganizationKind {
			continue
		}

		var organization v1alpha1.Organization

		objectKey := client.ObjectKey{
			Name:      ownerReference.Name,
			Namespace: cluster.Namespace,
		}

		err := c.Get(ctx, objectKey, &organization)
		if err != nil {
			return nil, err
		}

		return &organization, nil
	}

	return nil, nil
}

func NewClusterController(manager ctrl.Manager, clusterService clusterservices.ClusterService, logger *slog.Logger) error {
	client := manager.GetClient()

	c := clusterController{
		Client:         client,
		clusterService: clusterService,
		logger:         logger,
	}

	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.Cluster{}).Complete(&c)
	if err != nil {
		return err
	}

	return nil
}
