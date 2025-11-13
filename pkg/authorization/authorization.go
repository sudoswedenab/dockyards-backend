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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReconcileOrganizationSuperUserClusterRole(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + ":super-user",
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
		clusterRole.Labels = map[string]string{
			dockyardsv1.LabelOrganizationName: organization.Name,
		}

		clusterRole.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"delete",
					"patch",
					"update",
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

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + ":super-users",
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

		subjects := []rbacv1.Subject{}

		var memberList dockyardsv1.MemberList
		err = c.List(ctx, &memberList, client.InNamespace(organization.Spec.NamespaceRef.Name))
		if err != nil {
			return err
		}

		for _, member := range memberList.Items {
			if member.Spec.Role != dockyardsv1.RoleSuperUser {
				continue
			}

			subjects = append(subjects, rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     member.Spec.UserRef.Name,
			})
		}

		clusterRoleBinding.Subjects = subjects

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func ReconcileOrganizationReaderClusterRole(ctx context.Context, c client.Client, organization *dockyardsv1.Organization) error {
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + ":reader",
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
		clusterRole.Labels = map[string]string{
			dockyardsv1.LabelOrganizationName: organization.Name,
		}

		clusterRole.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

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

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:" + organization.Name + ":readers",
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, c, &clusterRoleBinding, func() error {
		clusterRoleBinding.Labels = map[string]string{
			dockyardsv1.LabelOrganizationName: organization.Name,
		}

		clusterRoleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: clusterRole.Name,
		}

		var memberList dockyardsv1.MemberList
		err = c.List(ctx, &memberList, client.InNamespace(organization.Spec.NamespaceRef.Name))
		if err != nil {
			return err
		}

		subjects := make([]rbacv1.Subject, len(memberList.Items))

		for i, member := range memberList.Items {
			subjects[i] = rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     member.Spec.UserRef.Name,
			}
		}

		clusterRoleBinding.Subjects = subjects

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
			Name: "dockyards:" + organization.Name + ":other",
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
		clusterRole.Labels = map[string]string{
			dockyardsv1.LabelOrganizationName: organization.Name,
		}

		clusterRole.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

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
			Name: "dockyards:" + organization.Name + ":others",
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, c, &clusterRoleBinding, func() error {
		clusterRoleBinding.Labels = map[string]string{
			dockyardsv1.LabelOrganizationName: organization.Name,
		}

		clusterRoleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.OrganizationKind,
				Name:       organization.Name,
				UID:        organization.UID,
			},
		}

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
	err := ReconcileOrganizationSuperUserClusterRole(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileOrganizationReaderClusterRole(ctx, c, organization)
	if err != nil {
		return err
	}

	err = ReconcileOtherUserClusterRoleAndBindings(ctx, c, organization)
	if err != nil {
		return err
	}

	return nil
}

func ReconcileClusterAuthorization(ctx context.Context, client client.Client) error {
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:reader",
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, client, &clusterRole, func() error {
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
					"clusters",
					"dnszones",
					"invitations",
					"members",
					"nodepools",
					"nodes",
					"workloads",
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	clusterRole = rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:user",
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, client, &clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"create",
					"delete",
					"patch",
					"update",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"clusters",
					"nodepools",
					"nodes",
					"workloads",
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	clusterRole = rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:super-user",
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, client, &clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"create",
					"delete",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"invitations",
				},
			},
			{
				Verbs: []string{
					"delete",
					"patch",
					"update",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"members",
				},
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func ReconcileMemberAuthorization(ctx context.Context, client client.Client, member *dockyardsv1.Member) error {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      member.Name + ":self",
			Namespace: member.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, client, &role, func() error {
		role.Labels = map[string]string{
			dockyardsv1.LabelMemberName: member.Name,
		}

		role.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.MemberKind,
				Name:       member.Name,
				UID:        member.UID,
			},
		}

		role.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"delete",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"members",
				},
				ResourceNames: []string{
					"@me",
					member.Name,
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
			Name:      member.Name + ":self",
			Namespace: member.Namespace,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, client, &roleBinding, func() error {
		roleBinding.Labels = map[string]string{
			dockyardsv1.LabelMemberName: member.Name,
		}

		roleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.MemberKind,
				Name:       member.Name,
				UID:        member.UID,
			},
		}

		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     role.Name,
		}

		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     rbacv1.UserKind,
				Name:     member.Spec.UserRef.Name,
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	roleBinding = rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      member.Name + ":reader",
			Namespace: member.Namespace,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, client, &roleBinding, func() error {
		roleBinding.Labels = map[string]string{
			dockyardsv1.LabelMemberName: member.Name,
		}

		roleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.MemberKind,
				Name:       member.Name,
				UID:        member.UID,
			},
		}

		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     "dockyards:reader",
		}

		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     rbacv1.UserKind,
				Name:     member.Spec.UserRef.Name,
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	if member.Spec.Role == dockyardsv1.RoleReader {
		return nil
	}

	roleBinding = rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      member.Name + ":user",
			Namespace: member.Namespace,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, client, &roleBinding, func() error {
		roleBinding.Labels = map[string]string{
			dockyardsv1.LabelMemberName: member.Name,
		}

		roleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.MemberKind,
				Name:       member.Name,
				UID:        member.UID,
			},
		}

		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     "dockyards:user",
		}

		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     rbacv1.UserKind,
				Name:     member.Spec.UserRef.Name,
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	if member.Spec.Role == dockyardsv1.RoleUser {
		return nil
	}

	roleBinding = rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      member.Name + ":super-user",
			Namespace: member.Namespace,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, client, &roleBinding, func() error {
		roleBinding.Labels = map[string]string{
			dockyardsv1.LabelMemberName: member.Name,
		}

		roleBinding.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.MemberKind,
				Name:       member.Name,
				UID:        member.UID,
			},
		}

		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     "dockyards:super-user",
		}

		roleBinding.Subjects = []rbacv1.Subject{
			{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     rbacv1.UserKind,
				Name:     member.Spec.UserRef.Name,
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func ReconcileUserAuthorization(ctx context.Context, c client.Client, user *dockyardsv1.User) error {
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dockyards:user:" + user.Name,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &clusterRole, func() error {
		clusterRole.Labels = map[string]string{
			dockyardsv1.LabelUserName: user.Name,
		}

		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"delete",
					"get",
					"patch",
					"update",
				},
				APIGroups: []string{
					dockyardsv1.GroupVersion.Group,
				},
				Resources: []string{
					"users",
				},
				ResourceNames: []string{
					user.Name,
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
			Name: "dockyards:user:" + user.Name,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, c, &clusterRoleBinding, func() error {
		clusterRoleBinding.Labels = map[string]string{
			dockyardsv1.LabelUserName: user.Name,
		}

		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Name:     user.Name,
			},
		}

		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
