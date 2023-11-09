package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func (o *Organization) SetupWebhookWithManager(manager ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(manager).For(o).Complete()
}
