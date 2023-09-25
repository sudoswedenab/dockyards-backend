package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetNodePool(c *gin.Context) {
	ctx := context.Background()

	nodePoolID := c.Param("nodePoolID")

	nodePool, err := h.clusterService.GetNodePool(nodePoolID)
	if err != nil {
		h.logger.Error("error getting node pool from cluster service", "id", nodePoolID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster, err := h.clusterService.GetCluster(nodePool.ClusterID)
	if err != nil {
		h.logger.Error("error getting node pool cluster", "id", nodePool.ClusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	objectKey := client.ObjectKey{
		Name: cluster.Organization,
	}

	var organization v1alpha1.Organization
	err = h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting node pool organization", "name", cluster.Organization, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("user is not a member of organization")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.JSON(http.StatusOK, nodePool)
}

func (h *handler) PostClusterNodePools(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")

	h.logger.Debug("param", "id", clusterID)

	var nodePoolOptions v1.NodePoolOptions
	err := c.BindJSON(&nodePoolOptions)
	if err != nil {
		h.logger.Error("failed bind node pool options from request body", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	details, isValidName := name.IsValidName(nodePoolOptions.Name)
	if !isValidName {
		h.logger.Error("node pool has invalid name", "name", nodePoolOptions.Name)

		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"details": details,
		})
		return
	}

	if nodePoolOptions.Quantity > 9 {
		h.logger.Debug("quantity too large", "quantity", nodePoolOptions.Quantity)

		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error":    "node pool quota exceeded",
			"quantity": nodePoolOptions.Quantity,
			"details":  "quantity must be lower than 9",
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

	objectKey := client.ObjectKey{
		Name: cluster.Organization,
	}

	var organization v1alpha1.Organization
	err = h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting organization from kubernetes", "name", cluster.Organization, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Error("subject is not a member of organization")

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

	v1Organization := v1.Organization{
		ID:   string(organization.UID),
		Name: organization.Name,
	}

	nodePool, err := h.clusterService.CreateNodePool(&v1Organization, cluster, &nodePoolOptions)
	if err != nil {
		h.logger.Error("error creating node pool in cluster service", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.JSON(http.StatusCreated, nodePool)
}

func (h *handler) DeleteNodePool(c *gin.Context) {
	ctx := context.Background()

	nodePoolID := c.Param("nodePoolID")

	nodePool, err := h.clusterService.GetNodePool(nodePoolID)
	if err != nil {
		h.logger.Error("error getting node pool from cluster service", "id", nodePoolID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster, err := h.clusterService.GetCluster(nodePool.ClusterID)
	if err != nil {
		h.logger.Error("error getting node pool cluster", "id", nodePool.ClusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	objectKey := client.ObjectKey{
		Name: cluster.Organization,
	}

	var organization v1alpha1.Organization
	err = h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting node pool organization", "name", cluster.Organization, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	v1Organization := v1.Organization{
		ID:   string(organization.UID),
		Name: organization.Name,
	}

	err = h.clusterService.DeleteNodePool(&v1Organization, nodePoolID)
	if err != nil {
		h.logger.Error("error deleting node pool", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
