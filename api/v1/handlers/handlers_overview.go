package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetOverview(c *gin.Context) {
	ctx := context.Background()

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizationList v1alpha1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList)
	if err != nil {
		h.logger.Error("error listing organizations in kubernetes", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var overview v1.Overview

	allClusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("error getting all clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	clustersOverviews := make(map[string][]v1.ClusterOverview)
	for _, cluster := range *allClusters {
		clusterOverview := v1.ClusterOverview{
			Name: cluster.Name,
			Id:   cluster.Id,
		}

		var deployments []v1.Deployment
		err := h.db.Find(&deployments, "cluster_id = ?", cluster.Id).Error
		if err != nil {
			h.logger.Error("error getting deployments from database", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
		}

		deploymentsOverview := []v1.DeploymentOverview{}

		for _, deployment := range deployments {
			deploymentOverview := v1.DeploymentOverview{
				Name: *deployment.Name,
				Id:   deployment.Id.String(),
			}

			deploymentsOverview = append(deploymentsOverview, deploymentOverview)
		}

		clusterOverview.Deployments = &deploymentsOverview

		organizationClusters := clustersOverviews[cluster.Organization]
		organizationClusters = append(organizationClusters, clusterOverview)
		clustersOverviews[cluster.Organization] = organizationClusters

	}

	for _, organization := range organizationList.Items {
		if !h.isMember(subject, &organization) {
			continue
		}

		clustersOverview := clustersOverviews[organization.Name]

		organizationOverview := v1.OrganizationOverview{
			Name:     organization.Name,
			Id:       string(organization.UID),
			Clusters: &clustersOverview,
		}

		usersOverview := []v1.UserOverview{}

		for _, memberRef := range organization.Spec.MemberRefs {
			h.logger.Debug("member", "name", memberRef.Name)

			objectKey := client.ObjectKey{
				Name:      memberRef.Name,
				Namespace: organization.Namespace,
			}

			var user v1alpha1.User
			err := h.controllerClient.Get(ctx, objectKey, &user)
			if err != nil {
				h.logger.Warn("error getting user from kubernetes", "err", err)

				continue
			}

			userOverview := v1.UserOverview{
				Id:    string(user.UID),
				Email: user.Spec.Email,
			}

			usersOverview = append(usersOverview, userOverview)
		}

		organizationOverview.Users = &usersOverview

		overview.Organizations = append(overview.Organizations, organizationOverview)
	}

	c.JSON(http.StatusOK, overview)
}
