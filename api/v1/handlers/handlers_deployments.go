package handlers

import (
	"errors"
	"net/http"
	"os"
	"path"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	utildeployment "bitbucket.org/sudosweden/dockyards-backend/pkg/util/deployment"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *handler) PostClusterDeployments(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		h.logger.Debug("cluster empty")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var deployment v1.Deployment
	err := c.BindJSON(&deployment)
	if err != nil {
		h.logger.Error("failed to read body", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	deployment.ClusterID = clusterID

	err = utildeployment.AddNormalizedName(&deployment)
	if err != nil {
		h.logger.Error("error adding deployment name", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	details, validName := names.IsValidName(*deployment.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    deployment.Name,
			"details": details,
		})
		return
	}

	var existingDeployment v1.Deployment
	err = h.db.Take(&existingDeployment, "name = ? AND cluster_id = ?", *deployment.Name, deployment.ClusterID).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Error("error taking deployment from database", "name", *deployment.Name, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.Debug("deployment name already in-use", "name", *deployment.Name, "cluster", deployment.ClusterID)

		c.AbortWithStatus(http.StatusConflict)
		return
	}

	err = utildeployment.CreateRepository(&deployment, h.gitProjectRoot)
	if err != nil {
		h.logger.Error("error creating deployment", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	err = h.db.Create(&deployment).Error
	if err != nil {
		h.logger.Error("error creating deployment in database", "name", deployment.Name, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	deploymentStatus := v1.DeploymentStatus{
		ID:           uuid.New(),
		DeploymentID: deployment.ID,
		State:        util.Ptr("pending"),
		Health:       util.Ptr(v1.DeploymentStatusHealthWarning),
	}

	err = h.db.Create(&deploymentStatus).Error
	if err != nil {
		h.logger.Warn("error creating deployment status in database", "err", err)
	}

	deployment.Status = deploymentStatus

	c.JSON(http.StatusCreated, deployment)
}

func (h *handler) GetClusterDeployments(c *gin.Context) {
	clusterID := c.Param("clusterID")

	cluster, err := h.clusterService.GetCluster(clusterID)
	if err != nil {
		h.logger.Error("error getting deployment cluster from cluster service", "id", clusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	h.logger.Debug("cluster", "organization", cluster.Organization)

	var deployments []v1.Deployment
	err = h.db.Find(&deployments, "cluster_id = ?", clusterID).Error
	if err != nil {
		h.logger.Error("error finding deployments in database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, deployments)
}

func (h *handler) DeleteDeployment(c *gin.Context) {
	deploymentID := c.Param("deploymentID")
	if deploymentID == "" {
		h.logger.Debug("deployment id empty")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var deploymentStatuses []v1.DeploymentStatus
	err := h.db.Find(&deploymentStatuses, "deployment_id = ?", deploymentID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.Error("error finding deployment statuses in database", "id", deploymentID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	for _, deploymentStatus := range deploymentStatuses {
		h.logger.Debug("deleting deployment status from database", "id", deploymentStatus.ID)

		err := h.db.Delete(&deploymentStatus).Error
		if err != nil {
			h.logger.Error("error deleting deployment status from database", "id", deploymentStatus.ID, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		h.logger.Debug("deleted deployment status from database", "id", deploymentStatus.ID)
	}

	var deployment v1.Deployment
	err = h.db.Take(&deployment, "id = ?", deploymentID).Error
	if err != nil {
		h.logger.Error("error taking deployment from database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleting deployment from database", "id", deployment.ID)

	err = h.db.Delete(&deployment).Error
	if err != nil {
		h.logger.Error("error deleting deployment from database", "id", deployment.ID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleted deployment from database", "id", deployment.ID)

	repoPath := path.Join(h.gitProjectRoot, "/v1/deployments", deployment.ID.String())

	h.logger.Debug("deleting repository from filesystem", "path", repoPath)

	err = os.RemoveAll(repoPath)
	if err != nil {
		h.logger.Warn("error deleting repository from filesystem", "path", repoPath, "err", err)
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("deploymentID")

	var deployment v1.Deployment
	err := h.db.Take(&deployment, "id = ?", deploymentID).Error
	if err != nil {
		h.logger.Debug("error taking deployment from database", "id", deploymentID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var deploymentStatus v1.DeploymentStatus
	err = h.db.Order("created_at desc").First(&deploymentStatus, "deployment_id = ?", deploymentID).Error
	if err != nil {
		h.logger.Warn("error taking deployment status from database", "id", deploymentID, "err", err)
	}

	deployment.Status = deploymentStatus

	c.JSON(http.StatusOK, deployment)
}
