package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *handler) GetCredentials(c *gin.Context) {
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
		h.logger.Error("error getting organizations from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	orgs := make(map[string]*v1alpha1.Organization)
	for i, organization := range organizationList.Items {
		orgs[organization.Name] = &organizationList.Items[i]
	}

	var credentials []v1.Credential
	err = h.db.Select("id", "name", "organization").Find(&credentials).Error
	if err != nil {
		h.logger.Error("error finding credentials in database", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	filteredCredentials := []v1.Credential{}

	for _, credential := range credentials {
		isMember := h.isMember(subject, orgs[credential.Organization])
		if isMember {
			filteredCredentials = append(filteredCredentials, credential)
		}
	}

	c.JSON(http.StatusOK, filteredCredentials)
}

func (h *handler) PostOrgCredentials(c *gin.Context) {
	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var credential v1.Credential
	err := c.BindJSON(&credential)
	if err != nil {
		h.logger.Error("error binding request json to credential", "err", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	credential.Id = uuid.New()
	credential.Organization = org

	err = h.db.Create(&credential).Error
	if err != nil {
		h.logger.Error("error creating credential in database", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, credential)
}

func (h *handler) DeleteOrgCredentials(c *gin.Context) {
	org := c.Param("org")
	name := c.Param("credential")

	var credential v1.Credential
	err := h.db.Take(&credential, "organization = ? and name = ?", org, name).Error
	if err != nil {
		h.logger.Error("error taking credential from database", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("delete credential", "id", credential.Id)

	err = h.db.Delete(&credential).Error
	if err != nil {
		h.logger.Error("error deleting credential from database", "id", credential.Id, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) GetCredentialUUID(c *gin.Context) {
	id := c.Param("uuid")

	var credential v1.Credential
	err := h.db.Take(&credential, "id = ?", id).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, credential)
}
