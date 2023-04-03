package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/gin-gonic/gin"
)

func (h *handler) ContainerOptions(c *gin.Context) {
	supportedVersions := h.clusterService.GetSupportedVersions()
	c.JSON(http.StatusOK, model.ContainerOptions{

		Options: []model.Options{{Name: "",
			SingleNode:      false,
			KubeVersion:     supportedVersions,
			NodePoolOptions: []model.NodePoolOptions{{}},
		}},
	})
}
