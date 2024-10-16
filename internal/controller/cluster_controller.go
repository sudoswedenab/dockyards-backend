package controller

import (
	"context"
	"slices"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	semverv3 "github.com/Masterminds/semver/v3"
	"github.com/fluxcd/pkg/runtime/patch"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;delete;list;patch;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

type ClusterReconciler struct {
	client.Client
	DockyardsNamespace string
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)

	var cluster dockyardsv1.Cluster
	err := r.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if apiutil.HasExpired(&cluster) && !cluster.Spec.BlockDeletion {
		logger.Info("deleting expired cluster")

		err := r.Delete(ctx, &cluster, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	var release dockyardsv1.Release
	err = r.Get(ctx, client.ObjectKey{Name: dockyardsv1.ReleaseNameSupportedKubernetesVersions, Namespace: r.DockyardsNamespace}, &release)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if apierrors.IsNotFound(err) {
		logger.Info("ignoring cluster with missing release")

		return ctrl.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(&cluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := patchHelper.Patch(ctx, &cluster)
		if err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	currentVersion, err := semverv3.NewVersion(cluster.Spec.Version)
	if err != nil {
		logger.Error(err, "error parsing current cluster version as semver")

		return ctrl.Result{}, nil
	}

	nextMinor := currentVersion.IncMinor()
	maxVersionSkew := nextMinor.IncMinor()

	upgrades := []dockyardsv1.ClusterUpgrade{}

	for _, version := range release.Status.Versions {
		newVersion, err := semverv3.NewVersion(version)
		if err != nil {
			logger.Error(err, "error parsing version as semver")

			continue
		}

		if currentVersion.GreaterThan(newVersion) || currentVersion.Equal(newVersion) {
			continue
		}

		if newVersion.GreaterThan(&maxVersionSkew) {
			continue
		}

		upgrades = append(upgrades, dockyardsv1.ClusterUpgrade{
			To: version,
		})
	}

	if !slices.Equal(cluster.Spec.Upgrades, upgrades) {
		logger.Info("new upgrades available", "old", cluster.Spec.Upgrades, "new", upgrades)

		cluster.Spec.Upgrades = upgrades
	}

	expiration := cluster.GetExpiration()
	cluster.Status.ExpirationTimestamp = expiration

	if expiration != nil {
		requeueAfter := expiration.Sub(time.Now())

		logger.Info("requeuing cluster until expiration", "expiration", expiration, "after", requeueAfter)

		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(mgr).For(&dockyardsv1.Cluster{}).Complete(r)
	if err != nil {
		return err
	}

	return nil
}
