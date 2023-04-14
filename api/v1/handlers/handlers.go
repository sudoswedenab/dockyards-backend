package handlers

import (
	"bitbucket.org/sudosweden/backend/api/v1/middleware"
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

	middlewareHandler := middleware.Handler{
		DB: db,
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
	g.POST("/logout", h.Logout)
	g.GET("/cluster-options", h.ContainerOptions)
	g.POST("/refresh", h.PostRefresh)

	g.POST("/clusters", h.PostClusters)
	g.GET("/clusters/:name/kubeconfig", h.GetClusterKubeConfig)
	g.GET("/clusters", h.GetClusters)
	g.DELETE("clusters/:name", h.DeleteCluster)
}
