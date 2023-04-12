package cluster

import (
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/types"
	"github.com/gin-gonic/gin"
)

type handler struct {
	clusterService   types.ClusterService
	accessTokenName  string
	refreshTokenName string
}

func RegisterRoutes(r *gin.Engine, clusterService types.ClusterService) {
	h := handler{
		clusterService:   clusterService,
		accessTokenName:  internal.AccessTokenName,
		refreshTokenName: internal.RefreshTokenName,
	}

	r.GET("/v1/clusters", h.GetAllClusters)
	r.DELETE("/v1/clusters/:name", h.DeleteCluster)
}
