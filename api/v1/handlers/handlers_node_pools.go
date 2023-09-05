package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
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

func (h *handler) PostClusterNodePools(c *gin.Context) {
	clusterID := c.Param("clusterID")

	h.logger.Debug("param", "id", clusterID)

	var nodePoolOptions v1.NodePoolOptions
	err := c.BindJSON(&nodePoolOptions)
	if err != nil {
		h.logger.Error("failed bind node pool options from request body", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	details, isValidName := names.IsValidName(nodePoolOptions.Name)
	if !isValidName {
		h.logger.Error("node pool has invalid name", "name", nodePoolOptions.Name)

		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"details": details,
		})
		return
	}

	cluster, err := h.clusterService.GetCluster(clusterID)
	if err != nil {
		h.logger.Error("error getting cluster from cluster service", "id", clusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	h.logger.Debug("got cluster from cluster service", "organization", cluster.Organization)

	var organization v1.Organization
	err = h.db.Take(&organization, "name = ?", cluster.Organization).Error
	if err != nil {
		h.logger.Error("error taking organization from database", "name", cluster.Organization, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Error("error getting user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember, err := h.isMember(&user, &organization)
	if err != nil {
		h.logger.Error("error getting user membership", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !isMember {
		h.logger.Error("user is not a member of organization")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	for _, nodePool := range cluster.NodePools {
		if nodePool.Name == nodePoolOptions.Name {
			h.logger.Error("node pool name already in-use", "id", nodePool.ID)

			c.AbortWithStatus(http.StatusConflict)
			return
		}
	}

	nodePool, err := h.clusterService.CreateNodePool(&organization, cluster, &nodePoolOptions)
	if err != nil {
		h.logger.Error("error creating node pool in cluster service", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.JSON(http.StatusCreated, nodePool)
}
