package cluster

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *handler) GetAllClusters(c *gin.Context) {
	// If filter len is 0, list all
	clusters, err := h.rancherService.GetAllClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"Clusters": clusters.Data,
	})
}
