package controller

import (
	"context"

	dockyardsv1alpha1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;patch;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations/status,verbs=patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=create;get;list;watch

func (r *OrganizationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	var organization dockyardsv1.Organization
	err := r.Get(ctx, req.NamespacedName, &organization)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !organization.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &organization)
	}

	if !controllerutil.ContainsFinalizer(&organization, OrganizationFinalizer) {
		patch := client.MergeFrom(organization.DeepCopy())

		controllerutil.AddFinalizer(&organization, OrganizationFinalizer)

		err := r.Patch(ctx, &organization, patch)
		if err != nil {
			logger.Error(err, "error adding finalizer")

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if organization.Status.NamespaceRef == "" {
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

		patch := client.MergeFrom(organization.DeepCopy())

		organization.Status.NamespaceRef = namespace.Name

		err = r.Status().Patch(ctx, &organization, patch)
		if err != nil {
			logger.Error(err, "error patching organization status")

			return ctrl.Result{}, err
		}

		logger.Info("created namespace for organization")

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileDelete(ctx context.Context, organization *dockyardsv1.Organization) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	matchingFields := client.MatchingFields{
		index.OwnerRefsIndexKey: string(organization.UID),
	}

	var clusterList dockyardsv1alpha1.ClusterList
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

func (r *OrganizationReconciler) SetupWithManager(manager ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(manager).
		For(&dockyardsv1.Organization{}).
		Owns(&dockyardsv1alpha1.Cluster{}).
		Complete(r)
}
