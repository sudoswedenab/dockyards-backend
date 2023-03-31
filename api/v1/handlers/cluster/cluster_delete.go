package cluster

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *handler) DeleteCluster(c *gin.Context) {
	clusterID := c.Param("id")

	err := h.rancherService.DeleteCluster(clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"Status": "Cluster Deleted",
	})
}
