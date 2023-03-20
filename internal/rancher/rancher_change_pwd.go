package rancher

import (
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) changeRancherPWD(user managementv3.User) (string, error) {
	RandomPwd := managementv3.SetPasswordInput{
		NewPassword: String(34),
	}

	changedUser, err := r.ManagementClient.User.ActionSetpassword(&user, &RandomPwd)
	r.Logger.Debug("response from set password action", "changedUser", changedUser)
	if err != nil {
		return "", err
	}

	return RandomPwd.NewPassword, nil
}
