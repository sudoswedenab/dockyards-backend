package cluster

import (
	"github.com/gin-gonic/gin"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"net/http"
)

func (h *handler) DeleteCluster(c *gin.Context) {
	clusterID := c.Param("id")
	cluster := managementv3.Cluster{UUID: clusterID}

	err := h.rancherService.DeleteCluster(cluster)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"Status": "Cluster Deleted",
	})
}
