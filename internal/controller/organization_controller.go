package controller

import (
	"context"

	dockyardsv1alpha1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	result, err := r.reconcileRoleBindings(ctx, &organization)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileRoleBindings(ctx context.Context, organization *dockyardsv1.Organization) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	superUsers := []types.UID{}
	users := []types.UID{}
	readers := []types.UID{}

	for _, memberRef := range organization.Spec.MemberRefs {
		switch memberRef.Role {
		case dockyardsv1.MemberRoleSuperUser:
			superUsers = append(superUsers, memberRef.UID)
			users = append(users, memberRef.UID)
			readers = append(readers, memberRef.UID)
		case dockyardsv1.MemberRoleUser:
			users = append(users, memberRef.UID)
			readers = append(readers, memberRef.UID)
		case dockyardsv1.MemberRoleReader:
			readers = append(readers, memberRef.UID)
		default:
			logger.Info("ignoring member reference with unsupported role", "role", memberRef.Role)
		}
	}

	result, err := r.reconcileSuperUserClusterRoleAndBinding(ctx, organization, superUsers)
	if err != nil {
		return result, err
	}

	result, err = r.reconcileUserRoleAndBinding(ctx, organization, users)
	if err != nil {
		return result, err
	}

	result, err = r.reconcileReaderClusterRoleAndBinding(ctx, organization, readers)
	if err != nil {
		return result, err
	}

	result, err = r.reconcileReaderRoleAndBinding(ctx, organization, readers)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileSuperUserClusterRoleAndBinding(ctx context.Context, organization *dockyardsv1.Organization, uids []types.UID) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards-" + organization.Name + "-superuser",
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &clusterRole, func() error {
		clusterRole.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		if clusterRole.Labels == nil {
			clusterRole.Labels = make(map[string]string)
		}

		clusterRole.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"*",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"organizations",
				},
				ResourceNames: []string{
					organization.Name,
				},
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrole", "clusterRoleName", clusterRole.Name, "result", operationResult)
	}

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards-" + organization.Name + "-superuser",
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, r.Client, &clusterRoleBinding, func() error {
		clusterRoleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		if clusterRoleBinding.Labels == nil {
			clusterRoleBinding.Labels = make(map[string]string)
		}

		clusterRoleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: clusterRole.Name,
		}

		subjects := make([]rbacv1.Subject, len(uids))

		for i, uid := range uids {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     string(uid),
			}
		}

		clusterRoleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrolebinding", "clusterRoleBindingName", clusterRoleBinding.Name, "result", operationResult)
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileUserRoleAndBinding(ctx context.Context, organization *dockyardsv1.Organization, uids []types.UID) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-user",
			Namespace: organization.Status.NamespaceRef,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &role, func() error {
		role.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		if role.Labels == nil {
			role.Labels = make(map[string]string)
		}

		role.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		role.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"delete",
					"patch",
					"watch",
				},
				APIGroups: []string{
					dockyardsv1alpha1.GroupVersion.Group,
				},
				Resources: []string{
					"*",
				},
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled role", "roleName", role.Name, "result", operationResult)
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-user",
			Namespace: organization.Status.NamespaceRef,
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, r.Client, &roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}

		roleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		}

		subjects := make([]rbacv1.Subject, len(uids))

		for i, uid := range uids {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     string(uid),
			}
		}

		roleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled rolebinding", "roleBindingName", roleBinding.Name, "result", operationResult)
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileReaderClusterRoleAndBinding(ctx context.Context, organization *dockyardsv1.Organization, uids []types.UID) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards-" + organization.Name + "-reader",
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &clusterRole, func() error {
		clusterRole.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		if clusterRole.Labels == nil {
			clusterRole.Labels = make(map[string]string)
		}

		clusterRole.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"organizations",
				},
				ResourceNames: []string{
					organization.Name,
				},
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrole", "clusterRoleName", clusterRole.Name, "result", operationResult)
	}

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards-" + organization.Name + "-reader",
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, r.Client, &clusterRoleBinding, func() error {
		clusterRoleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		if clusterRoleBinding.Labels == nil {
			clusterRoleBinding.Labels = make(map[string]string)
		}

		clusterRoleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: clusterRole.Name,
		}

		subjects := make([]rbacv1.Subject, len(uids))

		for i, uid := range uids {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     string(uid),
			}
		}

		clusterRoleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrolebinding", "clusterRoleBindingName", clusterRoleBinding.Name, "result", operationResult)
	}

	return ctrl.Result{}, nil
}

func (r *OrganizationReconciler) reconcileReaderRoleAndBinding(ctx context.Context, organization *dockyardsv1.Organization, uids []types.UID) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-reader",
			Namespace: organization.Status.NamespaceRef,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &role, func() error {
		role.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		if role.Labels == nil {
			role.Labels = make(map[string]string)
		}

		role.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		role.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{
					dockyardsv1alpha1.GroupVersion.Group,
				},
				Resources: []string{
					"*",
				},
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled role", "roleName", role.Name, "result", operationResult)
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-reader",
			Namespace: organization.Status.NamespaceRef,
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, r.Client, &roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}

		roleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		}

		subjects := make([]rbacv1.Subject, len(uids))

		for i, uid := range uids {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     string(uid),
			}
		}

		roleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled rolebinding", "roleBindingName", roleBinding.Name, "result", operationResult)
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

func (r *OrganizationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()

	_ = rbacv1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = dockyardsv1alpha1.AddToScheme(scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&dockyardsv1.Organization{}).
		Owns(&dockyardsv1alpha1.Cluster{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
