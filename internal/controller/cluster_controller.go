package controller

import (
	"context"
	"log/slog"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create

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

		c.logger.Debug("created cluster in cluster service", "id", createdCluster.Id)

		patch := client.MergeFrom(cluster.DeepCopy())

		cluster.Status.ClusterServiceID = createdCluster.Id
		err = c.Status().Patch(ctx, &cluster, patch)
		if err != nil {
			c.logger.Error("error patching cluster status", "err", err)

			return ctrl.Result{}, err
		}

		return requeue, nil
	}

	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-kubeconfig",
		Namespace: cluster.Namespace,
	}

	var secret corev1.Secret
	err = c.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		c.logger.Error("error getting cluster kubeconfig secret", "err", err)

		return ctrl.Result{}, err
	}

	if apierrors.IsNotFound(err) {
		c.logger.Debug("creating cluster kubeconfig")

		kubeconfig, err := c.clusterService.GetKubeconfig(cluster.Status.ClusterServiceID, 0)
		if err != nil {
			c.logger.Error("error getting cluster kubeconfig", "err", err)

			return ctrl.Result{}, err
		}

		value, err := clientcmd.Write(*kubeconfig)
		if err != nil {
			c.logger.Error("error serializing kubeconfig to yaml", "err", err)

			return ctrl.Result{}, err
		}

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-kubeconfig",
				Namespace: cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Data: map[string][]byte{
				"value": value,
			},
		}

		err = c.Create(ctx, &secret)
		if err != nil {
			c.logger.Error("error creating kubeconfig secret", "err", err)

			return ctrl.Result{}, err
		}
	}

	clusterStatus, err := c.clusterService.GetCluster(cluster.Status.ClusterServiceID)
	if err != nil {
		c.logger.Error("error getting cluster from cluster service", "err", err)

		return ctrl.Result{}, err
	}

	condition := meta.FindStatusCondition(clusterStatus.Conditions, v1alpha1.ReadyCondition)
	if condition == nil {
		c.logger.Debug("cluster service has not reported ready condition", "name", cluster.Name)

		return requeue, nil
	}

	patch := client.MergeFrom(cluster.DeepCopy())

	if cluster.Status.Version != clusterStatus.Version {
		cluster.Status.Version = clusterStatus.Version
	}

	meta.SetStatusCondition(&cluster.Status.Conditions, *condition)
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
			Id:   cluster.Status.ClusterServiceID,
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
