package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/gin-gonic/gin"
)

func (h *handler) getRecommendedNodePools() []model.NodePoolOptions {
	return []model.NodePoolOptions{
		{
			Name:                       "control-plane",
			ControlPlane:               true,
			Etcd:                       true,
			ControlPlaneComponentsOnly: true,
			Quantity:                   3,
			CPUCount:                   2,
			RAMSizeMB:                  4096,
			DiskSizeGB:                 10,
		},
		{
			Name:         "load-balancer",
			LoadBalancer: true,
			Quantity:     2,
			CPUCount:     2,
			RAMSizeMB:    4096,
			DiskSizeGB:   10,
		},
		{
			Name:       "worker",
			Quantity:   2,
			CPUCount:   4,
			RAMSizeMB:  8192,
			DiskSizeGB: 10,
		},
	}
}

func (h *handler) GetClusterOptions(c *gin.Context) {
	supportedVersions, err := h.clusterService.GetSupportedVersions()
	if err != nil {
		h.logger.Error("error getting supported versions from cluster service", "err", err)
	}

	recommendedNodePools := h.getRecommendedNodePools()

	c.JSON(http.StatusOK, model.Options{
		SingleNode:      false,
		Version:         supportedVersions,
		NodePoolOptions: recommendedNodePools,
	})
}
