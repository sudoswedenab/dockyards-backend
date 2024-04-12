package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetOrgs(c *gin.Context) {
	ctx := context.Background()

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)

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

		c.Status(http.StatusInternalServerError)
		return
	}

	var v1Organizations []v1.Organization
	for _, organization := range organizationList.Items {
		v1Organization := v1.Organization{
			Id:   string(organization.UID),
			Name: organization.Name,
		}

		v1Organizations = append(v1Organizations, v1Organization)
	}

	h.logger.Debug("organizations", "v1", v1Organizations)

	c.JSON(http.StatusOK, v1Organizations)
}
