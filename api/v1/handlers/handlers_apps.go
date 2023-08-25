package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"github.com/gin-gonic/gin"
)

func (h *handler) GetApps(c *gin.Context) {
	var apps []v1.App
	err := h.db.Find(&apps).Error
	if err != nil {
		h.logger.Error("error taking apps from database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.JSON(http.StatusOK, apps)
}

func (h *handler) GetApp(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		h.logger.Error("empty app id")

		c.AbortWithStatus(http.StatusBadRequest)
	}

	var app v1.App
	err := h.db.Preload("AppSteps.StepOptions").Preload("AppSteps").Take(&app, "id = ?", appID).Error
	if err != nil {
		h.logger.Error("error taking app from database", "id", appID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
	}

	c.JSON(http.StatusOK, app)
}
