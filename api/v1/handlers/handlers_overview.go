package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"github.com/gin-gonic/gin"
)

func (h *handler) GetOverview(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Error("error getting user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	var organizations []v1.Organization
	err = h.db.Joins("join organization_user on organization_user.organization_id = organizations.id").Where("organization_user.user_id = ?", user.ID).Find(&organizations).Error
	if err != nil {
		h.logger.Error("error finding organizations in database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	var overview v1.Overview

	allClusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("error getting all clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	clustersOverviews := make(map[string][]v1.ClusterOverview)
	for _, cluster := range *allClusters {
		clusterOverview := v1.ClusterOverview{
			Name: cluster.Name,
			ID:   cluster.ID,
		}

		var deployments []v1.Deployment
		err := h.db.Find(&deployments, "cluster_id = ?", cluster.ID).Error
		if err != nil {
			h.logger.Error("error getting deployments from database", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
		}

		deploymentsOverview := []v1.DeploymentOverview{}

		for _, deployment := range deployments {
			deploymentOverview := v1.DeploymentOverview{
				Name: deployment.Name,
				ID:   deployment.ID.String(),
			}

			deploymentsOverview = append(deploymentsOverview, deploymentOverview)
		}

		clusterOverview.Deployments = &deploymentsOverview

		organizationClusters := clustersOverviews[cluster.Organization]
		organizationClusters = append(organizationClusters, clusterOverview)
		clustersOverviews[cluster.Organization] = organizationClusters

	}

	for _, organization := range organizations {
		clustersOverview := clustersOverviews[organization.Name]

		organizationOverview := v1.OrganizationOverview{
			Name:     organization.Name,
			ID:       organization.ID.String(),
			Clusters: &clustersOverview,
		}

		var users []v1.User
		err := h.db.Debug().Joins("join organization_user on organization_user.user_id = users.id").Where("organization_user.organization_id = ?", organization.ID).Find(&users).Error
		if err != nil {
			h.logger.Error("error finding organization users in database", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
		}

		usersOverview := []v1.UserOverview{}

		for _, user := range users {
			h.logger.Debug("user", "organization", organization.ID, "id", user.ID)

			userOverview := v1.UserOverview{
				Email: user.Email,
				ID:    user.ID.String(),
			}

			usersOverview = append(usersOverview, userOverview)
		}

		organizationOverview.Users = &usersOverview

		overview.Organizations = append(overview.Organizations, organizationOverview)
	}

	c.JSON(http.StatusOK, overview)
}
