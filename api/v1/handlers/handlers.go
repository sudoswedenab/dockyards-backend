package handlers

import (
	"errors"
	"net/http"
	"net/http/cgi"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type handler struct {
	db                    *gorm.DB
	clusterService        types.ClusterService
	accessTokenName       string
	refreshTokenName      string
	logger                *slog.Logger
	jwtAccessTokenSecret  string
	jwtRefreshTokenSecret string
	flagServerCookie      bool
	cloudService          types.CloudService
}

type sudo struct {
	clusterService     types.ClusterService
	logger             *slog.Logger
	db                 *gorm.DB
	prometheusRegistry *prometheus.Registry
}

type HandlerOption func(*handler)

func WithCloudService(cloudService types.CloudService) HandlerOption {
	return func(h *handler) {
		h.cloudService = cloudService
	}
}

func WithJWTAccessTokens(accessToken, refreshToken string) HandlerOption {
	return func(h *handler) {
		h.jwtAccessTokenSecret = accessToken
		h.jwtRefreshTokenSecret = refreshToken
	}
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, clusterService types.ClusterService, logger *slog.Logger, accessTokenName, refreshTokenName string, flagServerCookie bool, handlerOptions ...HandlerOption) {
	methodNotAllowed := func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
	}

	h := handler{
		db:               db,
		clusterService:   clusterService,
		accessTokenName:  accessTokenName,
		refreshTokenName: refreshTokenName,
		logger:           logger,
		flagServerCookie: flagServerCookie,
	}

	for _, handlerOption := range handlerOptions {
		handlerOption(&h)
	}

	middlewareHandler := middleware.Handler{
		DB:                 db,
		Logger:             logger,
		AccessTokenSecret:  h.jwtAccessTokenSecret,
		RefreshTokenSecret: h.jwtRefreshTokenSecret,
		AccessTokenName:    accessTokenName,
		RefreshTokenName:   refreshTokenName,
	}

	gitHandler := cgi.Handler{
		Path: "/usr/lib/git-core/git-http-backend",
		Dir:  "/tmp/repos",
		Env: []string{
			"GIT_PROJECT_ROOT=/tmp/repos",
			"GIT_HTTP_EXPORT_ALL=true",
		},
	}

	anyGit := func(c *gin.Context) {
		git := c.Param("git")
		logger.Debug("git connection", "git", git)
		gitHandler.ServeHTTP(c.Writer, c.Request)
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
	g.POST("/logout", h.Logout)
	g.GET("/cluster-options", h.ContainerOptions)
	g.POST("/refresh", h.PostRefresh)

	g.GET("/clusters", h.GetClusters)

	g.GET("/orgs", h.GetOrgs)
	g.POST("orgs", h.PostOrgs)
	g.PUT("/orgs", methodNotAllowed)
	g.DELETE("/orgs", methodNotAllowed)

	g.POST("/orgs/:org/clusters", h.PostOrgClusters)
	g.DELETE("orgs/:org/clusters/:cluster", h.DeleteOrgClusters)
	g.GET("/orgs/:org/clusters/:cluster/kubeconfig", h.GetOrgClusterKubeConfig)

	r.POST("/v1/orgs/:org/clusters/:cluster/apps", h.PostOrgApps)
	r.GET("/v1/orgs/:org/clusters/:cluster/apps/*git", anyGit)
	r.POST("/v1/orgs/:org/clusters/:cluster/apps/*git", anyGit)
	r.DELETE("/v1/orgs/:org/clusters/:cluster/apps/:app", h.DeleteOrgApps)

	g.GET("/apps", h.GetApps)

	g.GET("/credentials", h.GetCredentials)
	g.GET("/credentials/:uuid", h.GetCredentialUUID)
	g.POST("/orgs/:org/credentials", h.PostOrgCredentials)
	g.DELETE("orgs/:org/credentials/:credential", h.DeleteOrgCredentials)
}

type SudoHandlerOption func(s *sudo)

func WithSudoClusterService(clusterService types.ClusterService) SudoHandlerOption {
	return func(s *sudo) {
		s.clusterService = clusterService
	}
}

func WithSudoLogger(logger *slog.Logger) SudoHandlerOption {
	return func(s *sudo) {
		s.logger = logger
	}
}

func WithSudoGormDB(db *gorm.DB) SudoHandlerOption {
	return func(s *sudo) {
		s.db = db
	}
}

func WithSudoPrometheusRegistry(registry *prometheus.Registry) SudoHandlerOption {
	return func(s *sudo) {
		s.prometheusRegistry = registry
	}
}

func RegisterSudoRoutes(e *gin.Engine, sudoHandlerOptions ...SudoHandlerOption) {
	s := sudo{}

	for _, sudoHandlerOption := range sudoHandlerOptions {
		sudoHandlerOption(&s)
	}

	e.GET("/sudo/clusters", s.GetClusters)
	e.GET("/sudo/kubeconfig/:org/:name", s.GetKubeconfig)
	e.GET("/sudo/apps", s.GetApps)
	e.GET("/sudo/orgs", s.GetOrgs)
	e.GET("/sudo/apps/:org/:cluster/:name", s.GetApp)

	handlerOpts := promhttp.HandlerOpts{
		Registry: s.prometheusRegistry,
	}

	handlerFor := promhttp.HandlerFor(s.prometheusRegistry, handlerOpts)

	e.GET("/metrics", func(c *gin.Context) {
		handlerFor.ServeHTTP(c.Writer, c.Request)
	})
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
