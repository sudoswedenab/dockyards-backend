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

	controlPlaneNodePoolOptions := model.NodePoolOptions{
		Name:                       "control-plane",
		Quantity:                   3,
		ControlPlane:               true,
		Etcd:                       true,
		ControlPlaneComponentsOnly: true,
	}

	controlPlaneNodePool, err := h.clusterService.CreateNodePool(cluster, &controlPlaneNodePoolOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	h.logger.Debug("created cluster control plane node pool", "name", controlPlaneNodePool.Name)

	for _, nodePoolOptions := range clusterOptions.NodePoolOptions {
		nodePool, err := h.clusterService.CreateNodePool(cluster, &nodePoolOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		h.logger.Debug("created cluster node pool", "name", nodePool.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":     "created successfully",
		"clusterName": cluster.Name,
	})
}
