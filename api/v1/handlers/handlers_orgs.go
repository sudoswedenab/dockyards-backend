package handlers

import (
	"errors"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handler) GetOrgs(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizations []model.Organization
	err = h.db.Model(&user).Association("Organizations").Find(&organizations)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, organizations)
}

func (h *handler) PostOrgs(c *gin.Context) {
	var organization model.Organization
	err := c.BindJSON(&organization)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	details, validName := internal.IsValidName(organization.Name)
	if !validName {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "name is not valid",
			"name":    organization.Name,
			"details": details,
		})
		return
	}

	var existingOrganization model.Organization
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
	organization.Users = []model.User{
		user,
	}

	err = h.db.Create(&organization).Error
	if err != nil {
		h.logger.Error("error addding organization to database", "name", organization.Name, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.Status(http.StatusOK)
}
