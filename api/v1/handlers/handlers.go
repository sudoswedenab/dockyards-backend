package handlers

import (
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/types"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type handler struct {
	db               *gorm.DB
	clusterService   types.ClusterService
	accessTokenName  string
	refreshTokenName string
	logger           *slog.Logger
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, clusterService types.ClusterService, logger *slog.Logger) {
	h := handler{
		db:               db,
		clusterService:   clusterService,
		accessTokenName:  internal.AccessTokenName,
		refreshTokenName: internal.RefreshTokenName,
		logger:           logger,
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)
	r.POST("/v1/logout", h.Logout)
	r.GET("/cluster-options", h.ContainerOptions)

	r.POST("/v1/clusters", h.PostClusters)
	r.GET("/v1/clusters/:name/kubeconfig", h.GetClusterKubeConfig)
}
