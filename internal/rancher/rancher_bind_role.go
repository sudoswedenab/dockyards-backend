package rancher

import (
	"fmt"

	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) BindRole(userid, roleName string) error {
	listOpts := types.ListOpts{
		Filters: map[string]interface{}{"Name": roleName},
	}
	globalRoleCollection, err := r.ManagementClient.GlobalRole.List(&listOpts)
	if err != nil {
		return err
	}

	var globalRole managementv3.GlobalRole
	for _, globalRoleCandidate := range globalRoleCollection.Data {
		if globalRoleCandidate.Name == roleName {
			globalRole = globalRoleCandidate
		}
	}

	globalRoleBinding := managementv3.GlobalRoleBinding{
		GlobalRoleID: globalRole.ID,
		UserID:       userid,
	}

	createdGlobalRoleBinding, err := r.ManagementClient.GlobalRoleBinding.Create(&globalRoleBinding)
	if err != nil {
		return err
	}

	fmt.Printf("create global role binding: %#v\n", createdGlobalRoleBinding)

	return nil
}
