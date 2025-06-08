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

package authorization

import (
	"context"

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReconcileUserRoleAndBindings(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	logger := ctrl.LoggerFrom(ctx)

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-user",
			Namespace: organization.Spec.NamespaceRef.Name,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, c, &role, func() error {
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
					"create",
					"delete",
					"patch",
					"watch",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"clusters",
					"workloads",
					"nodepools",
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled role", "roleName", role.Name, "result", operationResult)
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-users",
			Namespace: organization.Spec.NamespaceRef.Name,
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, c, &roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}

		roleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		}

		subjects := []rbacv1.Subject{}

		for _, memberRef := range organization.Spec.MemberRefs {
			if memberRef.Role == dockyardsv1.OrganizationMemberRoleReader {
				continue
			}

			subjects = append(subjects, rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     memberRef.Name,
			})
		}

		roleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled rolebinding", "roleBindingName", roleBinding.Name, "result", operationResult)
	}

	return nil
}

func ReconcileSuperUserClusterRoleAndBinding(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	logger := ctrl.LoggerFrom(ctx)

	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + "-superuser",
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
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
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrole", "clusterRoleName", clusterRole.Name, "result", operationResult)
	}

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + "-superusers",
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, c, &clusterRoleBinding, func() error {
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

		subjects := []rbacv1.Subject{}

		for _, memberRef := range organization.Spec.MemberRefs {
			if memberRef.Role != dockyardsv1.OrganizationMemberRoleSuperUser {
				continue
			}

			subjects = append(subjects, rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     memberRef.Name,
			})
		}

		clusterRoleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrolebinding", "clusterRoleBindingName", clusterRoleBinding.Name, "result", operationResult)
	}

	return nil
}

func ReconcileReaderClusterRoleAndBinding(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	logger := ctrl.LoggerFrom(ctx)

	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + "-reader",
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
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
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrole", "clusterRoleName", clusterRole.Name, "result", operationResult)
	}

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + "-readers",
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, c, &clusterRoleBinding, func() error {
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

		subjects := make([]rbacv1.Subject, len(organization.Spec.MemberRefs))

		for i, memberRef := range organization.Spec.MemberRefs {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     memberRef.Name,
			}
		}

		clusterRoleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled clusterrolebinding", "clusterRoleBindingName", clusterRoleBinding.Name, "result", operationResult)
	}

	return nil
}

func ReconcileReaderRoleAndBinding(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	logger := ctrl.LoggerFrom(ctx)

	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-reader",
			Namespace: organization.Spec.NamespaceRef.Name,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, c, &role, func() error {
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
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"*",
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled role", "roleName", role.Name, "result", operationResult)
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-readers",
			Namespace: organization.Spec.NamespaceRef.Name,
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, c, &roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}

		roleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		}

		subjects := make([]rbacv1.Subject, len(organization.Spec.MemberRefs))

		for i, memberRef := range organization.Spec.MemberRefs {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     memberRef.Name,
			}
		}

		roleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return err
	}

	if operationResult != controllerutil.OperationResultNone {
		logger.Info("reconciled rolebinding", "roleBindingName", roleBinding.Name, "result", operationResult)
	}

	return nil
}

func ReconcileSuperUserRoleAndBindings(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-superuser",
			Namespace: organization.Spec.NamespaceRef.Name,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &role, func() error {
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
					"create",
					"delete",
					"patch",
					"watch",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"invitations",
					"members",
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-superusers",
			Namespace: organization.Spec.NamespaceRef.Name,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, c, &roleBinding, func() error {
		if roleBinding.Labels == nil {
			roleBinding.Labels = make(map[string]string)
		}

		roleBinding.Labels[dockyardsv1.LabelOrganizationName] = organization.Name

		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "Role",
			Name: role.Name,
		}

		subjects := []rbacv1.Subject{}

		for _, memberRef := range organization.Spec.MemberRefs {
			if memberRef.Role != dockyardsv1.OrganizationMemberRoleSuperUser {
				continue
			}

			subjects = append(subjects, rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     memberRef.Name,
			})
		}

		roleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func ReconcileOtherUserClusterRoleAndBindings(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + "-other",
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
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
					"delete",
					"patch",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"invitations",
				},
				ResourceNames: []string{
					organization.Name,
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + "-others",
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, c, &clusterRoleBinding, func() error {
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

		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.GroupKind,
				Name:     "system:authenticated",
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func ReconcileOrganizationAuthorization(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	err := ReconcileSuperUserClusterRoleAndBinding(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileSuperUserRoleAndBindings(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileUserRoleAndBindings(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileReaderClusterRoleAndBinding(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileReaderRoleAndBinding(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileOtherUserClusterRoleAndBindings(ctx, c, organization)
	if err != nil {
		return err
	}

	return nil
}
