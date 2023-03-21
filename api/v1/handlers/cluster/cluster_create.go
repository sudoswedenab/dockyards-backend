package cluster

import (
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/gin-gonic/gin"
)

func (h *handler) CreateCluster(c *gin.Context) {
	var reqBody model.ClusterData
	if c.Bind(&reqBody) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	rancherCluster, err := h.rancherService.RancherCreateCluster(reqBody.DockerRootDir, reqBody.Name, reqBody.ClusterTemplateRevisionId, reqBody.ClusterTemplateId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	rancherNodePool, err := h.rancherService.RancherCreateNodePool(rancherCluster.ID, rancherCluster.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":     "created successfully",
		"clusterName": rancherNodePool.Name,
		"clusterId":   rancherNodePool.ClusterID,
	})
}
