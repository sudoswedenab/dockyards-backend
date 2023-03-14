package rancher

import (
	"fmt"
	"time"

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
		fmt.Printf("checking global role %s", globalRole.Name)
		if globalRole.Name == "dockyard-role" {
			fmt.Println(time.Now().Format(time.RFC822), " User role verified")
			create = false
		}
	}
	fmt.Printf("role 'dockyard-role' needs to be created: %t\n", create)
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
		fmt.Printf("role '%s' prepared with %d rules\n", globalRole.Name, len(globalRole.Rules))

		createdRole, err := r.ManagementClient.GlobalRole.Create(&globalRole)
		if err != nil {
			return err
		}

		fmt.Printf("role created: %#v\n", createdRole)

	}
	return nil
}
