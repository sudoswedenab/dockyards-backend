package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/gin-gonic/gin"
)

func (h *handler) GetClusterOptions(c *gin.Context) {
	supportedVersions := h.clusterService.GetSupportedVersions()

	c.JSON(http.StatusOK, model.Options{
		SingleNode: false,
		Version:    supportedVersions,
		NodePoolOptions: []model.NodePoolOptions{
			{
				Name:         "control-plane",
				ControlPlane: true,
				Etcd:         true,
				Quantity:     3,
				CPUCount:     2,
				RAMSizeMB:    4096,
				DiskSizeGB:   10,
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
		},
	})
}
