package rancher

import (
	"fmt"

	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) BindRole(userid, roleid string) error {
	globalRole, err := r.ManagementClient.GlobalRole.ByID(roleid)
	if err != nil {
		return fmt.Errorf("no role named '%s' found", roleid)
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
