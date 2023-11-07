package controller

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=releases/status,verbs=patch

var (
	releaseRequeue = ctrl.Result{RequeueAfter: time.Duration(time.Minute)}
)

type ReleaseReconciler struct {
	client.Client
	Logger         *slog.Logger
	ClusterService clusterservices.ClusterService
}

func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var release v1alpha1.Release
	err := r.Get(ctx, req.NamespacedName, &release)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error("error getting release", "err", err)

		return ctrl.Result{}, err
	}

	if !release.DeletionTimestamp.IsZero() {
		r.Logger.Debug("ignoring deleted release", "name", release.Name)

		return ctrl.Result{}, nil
	}

	switch release.Spec.Type {
	case v1alpha1.ReleaseTypeKubernetes:
		return r.reconcileKubernetesRelease(ctx, &release)
	default:
		r.Logger.Debug("ignoring release type", "type", release.Spec.Type)
	}

	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) reconcileKubernetesRelease(ctx context.Context, release *v1alpha1.Release) (ctrl.Result, error) {
	supportedVersions, err := r.ClusterService.GetSupportedVersions()
	if err != nil {
		r.Logger.Error("error getting supported versions", "err", err)

		return releaseRequeue, nil
	}

	if slices.Compare(supportedVersions, release.Status.Versions) != 0 {
		patch := client.MergeFrom(release.DeepCopy())

		release.Status.Versions = supportedVersions

		err := r.Status().Patch(ctx, release, patch)
		if err != nil {
			r.Logger.Error("error patching release", "err", err)

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) SetupWithManager(manager ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.Release{}).Complete(r)
	if err != nil {
		return err
	}

	return nil
}
