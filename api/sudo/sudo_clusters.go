package sudo

import (
	"context"
)

func (a *sudoAPI) GetClusters(ctx context.Context, req GetClustersRequestObject) (GetClustersResponseObject, error) {
	allClusters, err := a.clusterService.GetAllClusters()
	if err != nil {
		a.logger.Error("error getting clusters from cluster service", "err", err)

		return GetClusters500Response{}, nil
	}

	return GetClusters200JSONResponse(*allClusters), nil
}
