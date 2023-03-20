package rancher

import (
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) CreateClusterRole() error {
	globalRoleCollection, err := r.ManagementClient.GlobalRole.ListAll(&types.ListOpts{})
	if err != nil {
		return err
	}

	create := true
	for _, globalRole := range globalRoleCollection.Data {
		r.Logger.Debug("checking global role", "name", globalRole.Name)
		if globalRole.Name == "dockyard-role" {
			r.Logger.Debug("User role verified", "name", globalRole.Name)
			create = false
		}
	}
	r.Logger.Debug("role 'dockyard-role' info", "create", create)
	if create {
		globalRole := managementv3.GlobalRole{
			Name:           "dockyard-role",
			NewUserDefault: true,
			Rules: []managementv3.PolicyRule{
				{
					APIGroups: []string{"management.cattle.io"},
					Resources: []string{"nodetemplates"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{"management.cattle.io"},
					Resources: []string{"clustertemplaterevisions"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{"management.cattle.io"},
					Resources: []string{"nodepools"},
					Verbs:     []string{"update"},
				},
				{
					APIGroups: []string{"management.cattle.io"},
					Resources: []string{"clusters"},
					Verbs:     []string{"create", "list"},
				},
				{
					APIGroups: []string{"provisioning.cattle.io"},
					Resources: []string{"clusters"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{"management.cattle.io"},
					Resources: []string{"kontainerdrivers"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}
		r.Logger.Debug("role prepared", "name", globalRole.Name, "rules", len(globalRole.Rules))

		createdRole, err := r.ManagementClient.GlobalRole.Create(&globalRole)
		if err != nil {
			return err
		}

		r.Logger.Debug("role created", "createdRole", createdRole)

	}
	return nil
}
