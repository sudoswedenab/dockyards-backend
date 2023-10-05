package controller

import (
	"context"
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type organizationController struct {
	client.Client
	logger *slog.Logger
}

// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations/status,verbs=patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create

func (c *organizationController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var organization v1alpha1.Organization
	err := c.Get(ctx, req.NamespacedName, &organization)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if organization.Status.NamespaceRef == "" {
		c.logger.Info("organization has no namespace reference", "name", organization.Name)

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: organization.Name + "-",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &namespace)
		if err != nil {
			c.logger.Error("error creating namespace", "err", err)

			return ctrl.Result{}, err
		}

		patch := client.MergeFrom(organization.DeepCopy())
		organization.Status.NamespaceRef = namespace.Name

		err = c.Status().Patch(ctx, &organization, patch)
		if err != nil {
			c.logger.Error("error patching organization status", "err", err)

			return ctrl.Result{}, err
		}

		c.logger.Debug("created namespace for organization", "name", namespace.Name)

		return ctrl.Result{}, nil
	}

	c.logger.Debug("nothing to reconcile for organization", "name", organization.Name)

	return ctrl.Result{}, nil
}

func NewOrganizationController(manager ctrl.Manager, logger *slog.Logger) error {
	client := manager.GetClient()

	c := organizationController{
		Client: client,
		logger: logger,
	}

	err := ctrl.NewControllerManagedBy(manager).For(&v1alpha1.Organization{}).Complete(&c)
	if err != nil {
		return err
	}

	return nil
}
