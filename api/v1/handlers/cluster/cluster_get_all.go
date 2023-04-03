package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *handler) GetAllClusters(c *gin.Context) {
	// If filter len is 0, list all
	clusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusters": clusters,
	})
}
