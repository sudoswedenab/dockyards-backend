package sudo

import (
	"context"
)

func (a *sudoAPI) GetKubeconfig(ctx context.Context, req GetKubeconfigRequestObject) (GetKubeconfigResponseObject, error) {
	kubeconfig, err := a.clusterService.GetKubeconfig(req.ClusterID, 0)
	if err != nil {
		a.logger.Error("error getting kubeconfig", "err", err)

		return GetKubeconfig500Response{}, nil
	}

	return GetKubeconfig200JSONResponse(kubeconfig), nil

}
