package controller

import (
	"context"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/authorization"
	"github.com/fluxcd/pkg/runtime/patch"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	OrganizationFinalizer = "dockyards.io/backend-controller"
)

type OrganizationReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=dockyards.io,resources=*,verbs=*
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=create;get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=create;get;list;patch;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=create;get;list;patch;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;get;list;patch;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;get;list;patch;watch

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)

	var organization dockyardsv1.Organization
	err := r.Get(ctx, req.NamespacedName, &organization)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !organization.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &organization)
	}

	if apiutil.HasExpired(&organization) {
		logger.Info("organization has expired")

		err := r.Delete(ctx, &organization, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if apiutil.IgnoreInternalError(err) != nil {
			return ctrl.Result{}, err
		}

		if apierrors.IsInternalError(err) {
			logger.Info("ignoring internal error deleting expired cluster", "err", err)

			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		return ctrl.Result{}, nil
	}

	patchHelper, err := patch.NewHelper(&organization, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := patchHelper.Patch(ctx, &organization)
		if err != nil {
			result = ctrl.Result{}
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	if controllerutil.AddFinalizer(&organization, OrganizationFinalizer) {
		return ctrl.Result{}, nil
	}

	if organization.Status.NamespaceRef == nil {
		logger.Info("organization has no namespace reference")

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: organization.Name + "-",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
		}

		err := r.Create(ctx, &namespace)
		if err != nil {
			logger.Error(err, "error creating namespace")

			return ctrl.Result{}, err
		}

		organization.Status.NamespaceRef = &corev1.LocalObjectReference{
			Name: namespace.Name,
		}

		logger.Info("created namespace for organization")

		return ctrl.Result{}, nil
	}

	result, err = r.reconcileRoleBindings(ctx, &organization)
	if err != nil {
		return result, err
	}

	expiration := organization.GetExpiration()
	organization.Status.ExpirationTimestamp = expiration

	if expiration != nil {
		requeueAfter := expiration.Sub(time.Now())

		logger.Info("requeuing organization until expiration", "expiration", expiration, "after", requeueAfter)

		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileRoleBindings(ctx context.Context, organization *dockyardsv1.Organization) (ctrl.Result, error) {
	err := authorization.ReconcileSuperUserClusterRoleAndBinding(ctx, r.Client, organization)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = authorization.ReconcileUserRoleAndBindings(ctx, r.Client, organization)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = authorization.ReconcileReaderClusterRoleAndBinding(ctx, r.Client, organization)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = authorization.ReconcileReaderRoleAndBinding(ctx, r.Client, organization)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileDelete(ctx context.Context, organization *dockyardsv1.Organization) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	matchingFields := client.MatchingFields{
		index.OwnerReferencesField: string(organization.UID),
	}

	var clusterList dockyardsv1.ClusterList
	err := r.List(ctx, &clusterList, matchingFields)
	if err != nil {
		logger.Error(err, "error listing clusters")

		return ctrl.Result{}, err
	}

	if len(clusterList.Items) != 0 {
		logger.Info("ignoring deleted organization with clusters", "count", len(clusterList.Items))

		return ctrl.Result{}, nil
	}

	patch := client.MergeFrom(organization.DeepCopy())

	controllerutil.RemoveFinalizer(organization, OrganizationFinalizer)

	err = r.Patch(ctx, organization, patch)
	if err != nil {
		logger.Error(err, "error removing finalizer")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = rbacv1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&dockyardsv1.Organization{}).
		Owns(&dockyardsv1.Cluster{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
