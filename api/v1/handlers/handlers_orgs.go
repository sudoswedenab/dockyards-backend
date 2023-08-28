package handlers

import (
	"errors"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *handler) GetOrgs(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizations []v1.Organization
	err = h.db.Model(&user).Association("Organizations").Find(&organizations)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, organizations)
}

func (h *handler) PostOrgs(c *gin.Context) {
	var organization v1.Organization
	err := c.BindJSON(&organization)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	details, validName := names.IsValidName(organization.Name)
	if !validName {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "name is not valid",
			"name":    organization.Name,
			"details": details,
		})
		return
	}

	var existingOrganization v1.Organization
	err = h.db.Where("name = ?", organization.Name).Take(&existingOrganization).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Error("error checking for existing organization", "name", organization.Name, "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	if existingOrganization.Name == organization.Name {
		c.JSON(http.StatusConflict, gin.H{
			"error": "organization name is already in use, reserved or forbidden",
		})
		return
	}

	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// add the current user as the only user of the new organization
	// discard any users that might be part of the create request
	organization.Users = []v1.User{
		user,
	}

	organization.ID = uuid.New()

	err = h.db.Create(&organization).Error
	if err != nil {
		h.logger.Error("error addding organization to database", "name", organization.Name, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	cloudOrganizationID, err := h.cloudService.CreateOrganization(&organization)
	if err != nil {
		h.logger.Error("error creating organization in cloud service", "err", err)

		err = h.db.Select(clause.Associations).Delete(&organization).Error
		if err != nil {
			h.logger.Error("error deleting organization from database", "err", err)
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("created cloud organization", "id", cloudOrganizationID)

	c.Status(http.StatusOK)
}

func (h *handler) DeleteOrganization(c *gin.Context) {
	organizationID := c.Param("org")

	var organization v1.Organization
	err := h.db.Take(&organization, "id = ?", organizationID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		h.logger.Error("error taking organization from database", "id", organizationID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	allClusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("error getting all clusters from cluster service", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	organizationClusters := []v1.Cluster{}
	for _, cluster := range *allClusters {
		if cluster.Organization == organization.Name {
			h.logger.Warn("existing cluster belongs to organization", "organization", organization.ID, "id", cluster.ID)

			organizationClusters = append(organizationClusters, cluster)
		}
	}

	if len(organizationClusters) > 0 {
		h.logger.Debug("refusing delete since clusters exists in organization", "n", len(organizationClusters))

		c.JSON(http.StatusForbidden, organizationClusters)
		return
	}

	err = h.cloudService.DeleteOrganization(&organization)
	if err != nil {
		h.logger.Error("error deleting organization from cluster service", "id", organization.ID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	err = h.db.Model(&organization).Association("Users").Clear()
	if err != nil {
		h.logger.Error("error removing users from organization", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	err = h.db.Delete(&organization).Error
	if err != nil {
		h.logger.Error("error deleting organization from database", "id", organization.ID, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	h.logger.Debug("deleted organization from database", "id", organizationID)

	c.Status(http.StatusNoContent)
}
