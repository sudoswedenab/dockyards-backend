package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/gin-gonic/gin"
)

func (h *handler) PostClusters(c *gin.Context) {
	var clusterOptions model.ClusterOptions
	if c.BindJSON(&clusterOptions) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	cluster, err := h.clusterService.CreateCluster(&clusterOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = h.clusterService.CreateNodePool(cluster)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":     "created successfully",
		"clusterName": cluster.Name,
	})
}
