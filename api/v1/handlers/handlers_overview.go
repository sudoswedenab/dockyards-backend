package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/gin-gonic/gin"
)

func (h *handler) GetOverview(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Error("error getting user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	var organizations []model.Organization
	err = h.db.Joins("join organization_user on organization_user.organization_id = organizations.id").Where("organization_user.user_id = ?", user.ID).Find(&organizations).Error
	if err != nil {
		h.logger.Error("error finding organizations in database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	var overview model.Overview

	allClusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("error getting all clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	clustersOverviews := make(map[string][]model.ClusterOverview)
	for _, cluster := range *allClusters {
		clusterOverview := model.ClusterOverview{
			Name: cluster.Name,
			ID:   cluster.ID,
		}

		var deployments []model.Deployment
		err := h.db.Find(&deployments, "cluster_id = ?", cluster.ID).Error
		if err != nil {
			h.logger.Error("error getting deployments from database", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
		}

		for _, deployment := range deployments {
			deploymentOverview := model.DeploymentOverview{
				Name: deployment.Name,
				ID:   deployment.ID,
			}

			clusterOverview.Deployments = append(clusterOverview.Deployments, deploymentOverview)
		}

		organizationClusters := clustersOverviews[cluster.Organization]
		organizationClusters = append(organizationClusters, clusterOverview)
		clustersOverviews[cluster.Organization] = organizationClusters

	}

	for _, organization := range organizations {
		clustersOverview := clustersOverviews[organization.Name]

		organizationOverview := model.OrganizationOverview{
			Name:     organization.Name,
			ID:       organization.ID,
			Clusters: clustersOverview,
		}

		var users []model.User
		err := h.db.Debug().Joins("join organization_user on organization_user.user_id = users.id").Where("organization_user.organization_id = ?", organization.ID).Find(&users).Error
		if err != nil {
			h.logger.Error("error finding organization users in database", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
		}

		for _, user := range users {
			h.logger.Debug("user", "organization", organization.ID, "id", user.ID)

			userOverview := model.UserOverview{
				Email: user.Email,
				ID:    user.ID,
			}

			organizationOverview.Users = append(organizationOverview.Users, userOverview)
		}

		overview.Organizations = append(overview.Organizations, organizationOverview)
	}

	c.JSON(http.StatusOK, overview)
}
