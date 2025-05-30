// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"time"

	semverv3 "github.com/Masterminds/semver/v3"
	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/fluxcd/pkg/runtime/patch"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;delete;list;patch;watch
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

	result, err = r.reconcileClusterUpgrades(ctx, &cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	expiration := cluster.GetExpiration()
	cluster.Status.ExpirationTimestamp = expiration

	if expiration != nil {
		requeueAfter := time.Until(expiration.Time)

		logger.Info("requeuing cluster until expiration", "expiration", expiration, "after", requeueAfter)

		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) reconcileClusterUpgrades(ctx context.Context, dockyardsCluster *dockyardsv1.Cluster) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	if dockyardsCluster.Spec.Version == "" {
		return ctrl.Result{}, nil
	}

	featureEnabled, err := apiutil.IsFeatureEnabled(ctx, r, featurenames.FeatureClusterUpgrades, corev1.NamespaceAll)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !featureEnabled {
		dockyardsCluster.Spec.Upgrades = nil

		return ctrl.Result{}, nil
	}

	currentVersion, err := semverv3.NewVersion(dockyardsCluster.Spec.Version)
	if err != nil {
		conditions.MarkFalse(dockyardsCluster, dockyardsv1.ClusterUpgradesReadyCondition, dockyardsv1.ClusterUpgradesReconcileFailedReason, "%s", err)

		return ctrl.Result{}, nil
	}

	release, err := apiutil.GetDefaultRelease(ctx, r.Client, dockyardsv1.ReleaseTypeKubernetes)
	if err != nil {
		conditions.MarkFalse(dockyardsCluster, dockyardsv1.ClusterUpgradesReadyCondition, dockyardsv1.ClusterUpgradesReconcileFailedReason, "%s", err)

		return ctrl.Result{}, err
	}

	if release == nil {
		conditions.MarkFalse(dockyardsCluster, dockyardsv1.ClusterUpgradesReadyCondition, dockyardsv1.WaitingForDefaultReleaseReason, "")

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

	dockyardsCluster.Spec.Upgrades = upgrades

	conditions.MarkTrue(dockyardsCluster, dockyardsv1.ClusterUpgradesReadyCondition, dockyardsv1.ReadyReason, "")

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
