package jwt

import (
	"fmt"
	"net/http"

	"bitbucket.org/sudosweden/backend/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handler struct {
	db             *gorm.DB
	clusterService types.ClusterService
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, clusterService types.ClusterService) {
	h := handler{
		db:             db,
		clusterService: clusterService,
	}

	r.POST("/v1/refresh", func(c *gin.Context) {
		err := h.refreshTokenEndpoint(c)
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err))
		}
		c.String(http.StatusOK, "Success.")
	})
}
