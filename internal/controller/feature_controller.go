package controller

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/internal/feature"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=features,verbs=get;list;watch

type FeatureReconciler struct {
	client.Client
	DockyardsNamespace string
}

func (r *FeatureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	var dockyardsFeature dockyardsv1.Feature
	err := r.Get(ctx, req.NamespacedName, &dockyardsFeature)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !dockyardsFeature.DeletionTimestamp.IsZero() {
		feature.Disable(dockyardsv1.FeatureName(dockyardsFeature.Name))

		logger.Info("disabled feature")

		return ctrl.Result{}, nil
	}

	feature.Enable(dockyardsv1.FeatureName(dockyardsFeature.Name))

	logger.Info("enabled feature")

	return ctrl.Result{}, nil
}

func (r *FeatureReconciler) eventFilter() predicate.Funcs {
	return predicate.NewPredicateFuncs(func(o client.Object) bool {
		switch o.GetNamespace() {
		case r.DockyardsNamespace:
			return true
		default:
			return false
		}
	})
}

func (r *FeatureReconciler) SetupWithManager(m ctrl.Manager) error {
	scheme := m.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(m).
		For(&dockyardsv1.Feature{}).
		WithEventFilter(r.eventFilter()).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
