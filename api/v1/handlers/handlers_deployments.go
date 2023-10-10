package handlers

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	utildeployment "bitbucket.org/sudosweden/dockyards-backend/pkg/util/deployment"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	deployment.ClusterId = clusterID

	err = utildeployment.AddNormalizedName(&deployment)
	if err != nil {
		h.logger.Error("error adding deployment name", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	details, validName := name.IsValidName(*deployment.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    deployment.Name,
			"details": details,
		})
		return
	}

	var existingDeployment v1.Deployment
	err = h.db.Take(&existingDeployment, "name = ? AND cluster_id = ?", *deployment.Name, deployment.ClusterId).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Error("error taking deployment from database", "name", *deployment.Name, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.Debug("deployment name already in-use", "name", *deployment.Name, "cluster", deployment.ClusterId)

		c.AbortWithStatus(http.StatusConflict)
		return
	}

	if deployment.Type == v1.DeploymentTypeContainerImage || deployment.Type == v1.DeploymentTypeKustomize {
		err = utildeployment.CreateRepository(&deployment, h.gitProjectRoot)
		if err != nil {
			h.logger.Error("error creating deployment", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}

	err = h.db.Create(&deployment).Error
	if err != nil {
		h.logger.Error("error creating deployment in database", "name", deployment.Name, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	deploymentStatus := v1.DeploymentStatus{
		Id:           uuid.New(),
		DeploymentId: deployment.Id,
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
	ctx := context.Background()

	clusterID := c.Param("clusterID")

	matchingFields := client.MatchingFields{
		"metadata.uid": clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster := clusterList.Items[0]

	var deployments []v1.Deployment
	err = h.db.Find(&deployments, "cluster_id = ?", cluster.UID).Error
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
		h.logger.Debug("deleting deployment status from database", "id", deploymentStatus.Id)

		err := h.db.Delete(&deploymentStatus).Error
		if err != nil {
			h.logger.Error("error deleting deployment status from database", "id", deploymentStatus.Id, "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		h.logger.Debug("deleted deployment status from database", "id", deploymentStatus.Id)
	}

	var deployment v1.Deployment
	err = h.db.Take(&deployment, "id = ?", deploymentID).Error
	if err != nil {
		h.logger.Error("error taking deployment from database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleting deployment from database", "id", deployment.Id)

	err = h.db.Delete(&deployment).Error
	if err != nil {
		h.logger.Error("error deleting deployment from database", "id", deployment.Id, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleted deployment from database", "id", deployment.Id)

	repoPath := path.Join(h.gitProjectRoot, "/v1/deployments", deployment.Id.String())

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
