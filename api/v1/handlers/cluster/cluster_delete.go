package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *handler) DeleteCluster(c *gin.Context) {
	name := c.Param("name")

	err := h.clusterService.DeleteCluster(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Cluster Deleted",
	})
}
