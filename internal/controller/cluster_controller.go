package controller

import (
	"context"
	"log/slog"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
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
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create

var (
	requeue = ctrl.Result{Requeue: true, RequeueAfter: time.Duration(time.Minute * 2)}
)

const (
	ClusterFinalizer = "dockyards.io/backend-controller"
)

type ClusterReconciler struct {
	client.Client
	Logger         *slog.Logger
	ClusterService clusterservices.ClusterService
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With("name", req.Name, "namespace", req.Namespace)

	var cluster v1alpha1.Cluster
	err := r.Get(ctx, req.NamespacedName, &cluster)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting cluster", "err", err)

		return ctrl.Result{}, err
	}

	organization, err := apiutil.GetOwnerOrganization(ctx, r.Client, &cluster)
	if err != nil {
		logger.Error("error getting owner organization", "err", err)

		return ctrl.Result{}, err
	}

	if organization == nil {
		logger.Info("ignoring cluster without organization owner")

		return ctrl.Result{}, nil
	}

	if !cluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &cluster, organization)
	}

	if !controllerutil.ContainsFinalizer(&cluster, ClusterFinalizer) {
		patch := client.MergeFrom(cluster.DeepCopy())

		controllerutil.AddFinalizer(&cluster, ClusterFinalizer)

		err := r.Patch(ctx, &cluster, patch)
		if err != nil {
			logger.Error("error adding finalizer", "err", err)

			return ctrl.Result{}, err
		}
	}

	if cluster.Status.ClusterServiceID == "" {
		logger.Debug("cluster has empty cluster service id")

		clusterOptions := v1.ClusterOptions{
			Name: cluster.Name,
		}

		createdCluster, err := r.ClusterService.CreateCluster(organization, &clusterOptions)
		if err != nil {
			logger.Error("error creating cluster in cluster service", "err", err)

			return ctrl.Result{}, err
		}

		logger.Debug("created cluster in cluster service", "id", createdCluster.Id)

		patch := client.MergeFrom(cluster.DeepCopy())

		cluster.Status.ClusterServiceID = createdCluster.Id

		err = r.Status().Patch(ctx, &cluster, patch)
		if err != nil {
			logger.Error("error patching cluster status", "err", err)

			return ctrl.Result{}, err
		}

		return requeue, nil
	}

	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-kubeconfig",
		Namespace: cluster.Namespace,
	}

	var secret corev1.Secret
	err = r.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting cluster kubeconfig secret", "err", err)

		return ctrl.Result{}, err
	}

	if apierrors.IsNotFound(err) {
		logger.Debug("creating cluster kubeconfig")

		kubeconfig, err := r.ClusterService.GetKubeconfig(cluster.Status.ClusterServiceID, 0)
		if err != nil {
			logger.Error("error getting cluster kubeconfig", "err", err)

			return ctrl.Result{}, err
		}

		value, err := clientcmd.Write(*kubeconfig)
		if err != nil {
			logger.Error("error serializing kubeconfig to yaml", "err", err)

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

		err = r.Create(ctx, &secret)
		if err != nil {
			logger.Error("error creating kubeconfig secret", "err", err)

			return ctrl.Result{}, err
		}
	}

	clusterStatus, err := r.ClusterService.GetCluster(cluster.Status.ClusterServiceID)
	if err != nil {
		logger.Error("error getting cluster from cluster service", "err", err)

		return ctrl.Result{}, err
	}

	condition := meta.FindStatusCondition(clusterStatus.Conditions, v1alpha1.ReadyCondition)
	if condition == nil {
		logger.Debug("cluster service has not reported ready condition", "name", cluster.Name)

		return requeue, nil
	}

	patch := client.MergeFrom(cluster.DeepCopy())

	if cluster.Status.Version != clusterStatus.Version {
		cluster.Status.Version = clusterStatus.Version
	}

	meta.SetStatusCondition(&cluster.Status.Conditions, *condition)
	err = r.Status().Patch(ctx, &cluster, patch)
	if err != nil {
		logger.Error("error patching cluster status conditions", "err", err)

		return ctrl.Result{}, err
	}

	return requeue, nil
}

func (r *ClusterReconciler) reconcileDelete(ctx context.Context, cluster *v1alpha1.Cluster, organization *v1alpha2.Organization) (ctrl.Result, error) {
	logger := r.Logger.With("name", cluster.Name, "namespace", cluster.Namespace)

	matchingFields := client.MatchingFields{
		index.OwnerRefsIndexKey: string(cluster.UID),
	}

	var nodePoolList v1alpha1.NodePoolList
	err := r.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		logger.Error("error listing node pools", "err", err)

		return ctrl.Result{}, err
	}

	if len(nodePoolList.Items) != 0 {
		logger.Info("requeing deleted cluster with node pools", "count", len(nodePoolList.Items))

		return requeue, nil
	}

	if cluster.Status.ClusterServiceID == "" {
		logger.Warn("cluster has no cluster service id")
	}

	if cluster.Status.ClusterServiceID != "" {
		v1Cluster := v1.Cluster{
			Id:   cluster.Status.ClusterServiceID,
			Name: cluster.Name,
		}

		err := r.ClusterService.DeleteCluster(organization, &v1Cluster)
		if err != nil {
			logger.Error("error deleting cluster from cluster service", "err", err)

			return ctrl.Result{}, err
		}
	}

	patch := client.MergeFrom(cluster.DeepCopy())

	controllerutil.RemoveFinalizer(cluster, ClusterFinalizer)

	err = r.Patch(ctx, cluster, patch)
	if err != nil {
		logger.Error("error removing finalizer from cluster", "err", err)

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) SetupWithManager(manager ctrl.Manager) error {
	scheme := manager.GetScheme()
	v1alpha1.AddToScheme(scheme)
	v1alpha2.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(manager).
		For(&v1alpha1.Cluster{}).
		Owns(&v1alpha1.NodePool{}).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
