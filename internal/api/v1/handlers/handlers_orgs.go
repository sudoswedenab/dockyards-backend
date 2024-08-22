package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
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
		index.MemberReferencesField: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		h.logger.Error("error listing organizations in kubernetes", "err", err)
		c.Status(http.StatusInternalServerError)

		return
	}

	var organizations []v1.Organization
	for _, organization := range organizationList.Items {
		v1Organization := v1.Organization{
			Id:   string(organization.UID),
			Name: organization.Name,
		}

		organizations = append(organizations, v1Organization)
	}

	c.JSON(http.StatusOK, organizations)
}
