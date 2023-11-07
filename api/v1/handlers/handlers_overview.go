package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
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

	matchingFields := client.MatchingFields{
		index.MemberRefsIndexKey: subject,
	}

	var organizationList v1alpha1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList, matchingFields)
	if err != nil {
		h.logger.Error("error listing organizations in kubernetes", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var overview v1.Overview

	for _, organization := range organizationList.Items {
		organizationOverview := v1.OrganizationOverview{
			Name: organization.Name,
			Id:   string(organization.UID),
		}

		matchingFields := client.MatchingFields{
			index.OwnerRefsIndexKey: string(organization.UID),
		}

		var clusterList v1alpha1.ClusterList
		err := h.controllerClient.List(ctx, &clusterList, matchingFields)
		if err != nil {
			h.logger.Error("error listing clusters", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		clustersOverviews := []v1.ClusterOverview{}
		for _, cluster := range clusterList.Items {
			clusterOverview := v1.ClusterOverview{
				Name: cluster.Name,
				Id:   string(cluster.UID),
			}

			matchingFields := client.MatchingFields{
				index.OwnerRefsIndexKey: string(cluster.UID),
			}

			var deploymentList v1alpha1.DeploymentList
			err := h.controllerClient.List(ctx, &deploymentList, matchingFields)
			if err != nil {
				h.logger.Error("error listing deployments", "err", err)

				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			deploymentsOverview := []v1.DeploymentOverview{}
			for _, deployment := range deploymentList.Items {
				deploymentOverview := v1.DeploymentOverview{
					Name: deployment.Name,
					Id:   string(deployment.UID),
				}

				deploymentsOverview = append(deploymentsOverview, deploymentOverview)
			}

			if len(deploymentsOverview) != 0 {
				clusterOverview.Deployments = &deploymentsOverview
			}

			clustersOverviews = append(clustersOverviews, clusterOverview)
		}

		if len(clustersOverviews) != 0 {
			organizationOverview.Clusters = &clustersOverviews
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

		if len(usersOverview) != 0 {
			organizationOverview.Users = &usersOverview
		}

		overview.Organizations = append(overview.Organizations, organizationOverview)
	}

	c.JSON(http.StatusOK, overview)
}
