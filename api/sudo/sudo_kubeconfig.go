package sudo

import (
	"context"
)

func (a *sudoAPI) GetKubeconfig(ctx context.Context, req GetKubeconfigRequestObject) (GetKubeconfigResponseObject, error) {
	return GetKubeconfig500Response{}, nil
}
