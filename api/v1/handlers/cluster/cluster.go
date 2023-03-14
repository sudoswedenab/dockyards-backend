package cluster

import (
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/rancher"
	"github.com/gin-gonic/gin"
)

type handler struct {
	rancherService   rancher.RancherService
	accessTokenName  string
	refreshTokenName string
}

func RegisterRoutes(r *gin.Engine, rancherService rancher.RancherService) {
	h := handler{
		rancherService:   rancherService,
		accessTokenName:  internal.AccessTokenName,
		refreshTokenName: internal.RefreshTokenName,
	}

	r.POST("/v1/createcluster", h.CreateCluster)
}
