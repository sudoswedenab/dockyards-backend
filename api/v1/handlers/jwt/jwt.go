package jwt

import (
	"fmt"
	"net/http"

	"bitbucket.org/sudosweden/backend/internal/rancher"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handler struct {
	db             *gorm.DB
	rancherService rancher.RancherService
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rancherService rancher.RancherService) {
	h := handler{
		db:             db,
		rancherService: rancherService,
	}

	r.POST("/v1/refresh", func(c *gin.Context) {
		err := h.refreshTokenEndpoint(c)
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err))
		}
		c.String(http.StatusOK, "Success.")
	})
}
