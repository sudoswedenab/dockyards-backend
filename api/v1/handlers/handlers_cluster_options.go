package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/gin-gonic/gin"
)

func (h *handler) getRecommendedNodePools() []v1.NodePoolOptions {
	return []v1.NodePoolOptions{
		{
			Name:                       "control-plane",
			Quantity:                   3,
			ControlPlane:               util.Ptr(true),
			Etcd:                       util.Ptr(true),
			ControlPlaneComponentsOnly: util.Ptr(true),
			CPUCount:                   util.Ptr(2),
			RAMSizeMb:                  util.Ptr(4096),
			DiskSizeGb:                 util.Ptr(100),
		},
		{
			Name:                       "load-balancer",
			Quantity:                   2,
			LoadBalancer:               util.Ptr(true),
			ControlPlaneComponentsOnly: util.Ptr(true),
			CPUCount:                   util.Ptr(2),
			RAMSizeMb:                  util.Ptr(4096),
			DiskSizeGb:                 util.Ptr(100),
		},
		{
			Name:       "worker",
			Quantity:   2,
			CPUCount:   util.Ptr(4),
			RAMSizeMb:  util.Ptr(8192),
			DiskSizeGb: util.Ptr(100),
		},
	}
}

func (h *handler) GetClusterOptions(c *gin.Context) {
	supportedVersions, err := h.clusterService.GetSupportedVersions()
	if err != nil {
		h.logger.Error("error getting supported versions from cluster service", "err", err)
	}

	recommendedNodePools := h.getRecommendedNodePools()

	c.JSON(http.StatusOK, v1.Options{
		SingleNode:      false,
		Version:         supportedVersions,
		NodePoolOptions: recommendedNodePools,
	})
}
