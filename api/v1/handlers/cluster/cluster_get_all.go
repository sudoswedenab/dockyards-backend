package cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/rancher/norman/types"
	"net/http"
)

func (h *handler) GetAllClusters(c *gin.Context) {
	// If filter len is 0, list all
	opts := &types.ListOpts{}
	clusters, err := h.rancherService.GetClusters(opts)
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
