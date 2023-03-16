package rancher

import (
	"fmt"

	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) changeRancherPWD(user managementv3.User) (string, error) {
	RandomPwd := managementv3.SetPasswordInput{
		NewPassword: String(34),
	}

	changedUser, err := r.ManagementClient.User.ActionSetpassword(&user, &RandomPwd)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	fmt.Printf("changed user: %#v\n", changedUser)

	return RandomPwd.NewPassword, nil
}
