package handlers

import (
	"errors"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/middleware"
	"bitbucket.org/sudosweden/backend/api/v1/model"
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

type sudo struct {
	clusterService types.ClusterService
	logger         *slog.Logger
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, clusterService types.ClusterService, logger *slog.Logger) {
	methodNotAllowed := func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
	}

	h := handler{
		db:               db,
		clusterService:   clusterService,
		accessTokenName:  internal.AccessTokenName,
		refreshTokenName: internal.RefreshTokenName,
		logger:           logger,
	}

	middlewareHandler := middleware.Handler{
		DB:     db,
		Logger: logger,
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
	g.POST("/logout", h.Logout)
	g.GET("/cluster-options", h.ContainerOptions)
	g.POST("/refresh", h.PostRefresh)

	g.GET("/clusters/:name/kubeconfig", h.GetClusterKubeConfig)
	g.GET("/clusters", h.GetClusters)
	g.DELETE("clusters/:name", h.DeleteCluster)

	g.GET("/orgs", h.GetOrgs)
	g.POST("orgs", h.PostOrgs)
	g.PUT("/orgs", methodNotAllowed)
	g.DELETE("/orgs", methodNotAllowed)

	g.POST("/orgs/:org/clusters", h.PostOrgClusters)
}

func RegisterSudoRoutes(e *gin.Engine, clusterService types.ClusterService, logger *slog.Logger) {
	s := sudo{
		clusterService: clusterService,
		logger:         logger,
	}

	e.GET("/sudo/clusters", s.GetClusters)
}

func (h *handler) getUserFromContext(c *gin.Context) (model.User, error) {
	u, exists := c.Get("user")
	if !exists {
		return model.User{}, errors.New("error fecthing user from context")
	}

	user, ok := u.(model.User)
	if !ok {
		return model.User{}, errors.New("error during user type conversion")
	}

	return user, nil
}
