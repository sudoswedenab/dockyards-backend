package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *handler) GetNodePool(c *gin.Context) {
	nodePoolID := c.Param("nodePoolID")

	nodePool, err := h.clusterService.GetNodePool(nodePoolID)
	if err != nil {
		h.logger.Error("error getting node pool from cluster service", "id", nodePoolID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
	}

	c.JSON(http.StatusOK, nodePool)
}
