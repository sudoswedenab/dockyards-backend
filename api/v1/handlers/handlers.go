package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/http/cgi"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultDockyardsNamespace = "dockyards"
	defaultJWTSecretName      = "jwt-tokens"
)

type handler struct {
	db                    *gorm.DB
	clusterService        types.ClusterService
	accessTokenName       string
	refreshTokenName      string
	logger                *slog.Logger
	jwtAccessTokenSecret  string
	jwtRefreshTokenSecret string
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

func WithClusterService(clusterService types.ClusterService) HandlerOption {
	return func(h *handler) {
		h.clusterService = clusterService
	}
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *slog.Logger, handlerOptions ...HandlerOption) error {
	methodNotAllowed := func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
	}

	h := handler{
		db:               db,
		accessTokenName:  "AccessToken",
		refreshTokenName: "RefreshToken",
		logger:           logger,
	}

	for _, handlerOption := range handlerOptions {
		handlerOption(&h)
	}

	if h.jwtAccessTokenSecret == "" || h.jwtRefreshTokenSecret == "" {
		logger.Info("using jwt private secrets from kubernetes")

		err := h.setOrGenerateTokens()
		if err != nil {
			return err
		}
	}

	middlewareHandler := middleware.Handler{
		DB:                 db,
		Logger:             logger,
		AccessTokenSecret:  h.jwtAccessTokenSecret,
		RefreshTokenSecret: h.jwtRefreshTokenSecret,
		AccessTokenName:    "AccessToken",
		RefreshTokenName:   "RefreshToken",
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
	r.POST("/v1/refresh", h.PostRefresh)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
	g.POST("/logout", h.Logout)
	g.GET("/cluster-options", h.GetClusterOptions)

	g.GET("/clusters", h.GetClusters)
	g.GET("/clusters/:id", h.GetCluster)

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
	g.GET("/apps/:id", h.GetApp)

	g.GET("/credentials", h.GetCredentials)
	g.GET("/credentials/:uuid", h.GetCredentialUUID)
	g.POST("/orgs/:org/credentials", h.PostOrgCredentials)
	g.DELETE("orgs/:org/credentials/:credential", h.DeleteOrgCredentials)

	g.GET("/overview", h.GetOverview)

	return nil
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
	e.POST("/sudo/apps", s.PostApps)

	handlerOpts := promhttp.HandlerOpts{
		Registry: s.prometheusRegistry,
	}

	handlerFor := promhttp.HandlerFor(s.prometheusRegistry, handlerOpts)

	e.GET("/metrics", func(c *gin.Context) {
		handlerFor.ServeHTTP(c.Writer, c.Request)
	})

	e.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
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

func (h *handler) setOrGenerateTokens() error {
	ctx := context.Background()

	kubeconfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	controllerClient, err := client.New(kubeconfig, client.Options{})
	if err != nil {
		return err
	}

	objectKey := client.ObjectKey{
		Namespace: defaultDockyardsNamespace,
		Name:      defaultJWTSecretName,
	}

	var secret corev1.Secret
	err = controllerClient.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if apierrors.IsNotFound(err) {
		h.logger.Debug("generating private secrets")

		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			return err
		}
		accessToken := base64.StdEncoding.EncodeToString(b)

		h.logger.Debug("generated access token")

		b = make([]byte, 32)
		_, err = rand.Read(b)
		if err != nil {
			return err
		}
		refreshToken := base64.StdEncoding.EncodeToString(b)

		h.logger.Debug("generated refresh token")

		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultDockyardsNamespace,
				Name:      defaultJWTSecretName,
			},
			StringData: map[string]string{
				"accessToken":  accessToken,
				"refreshToken": refreshToken,
			},
		}

		err = controllerClient.Create(ctx, &secret)
		if err != nil {
			return err
		}

		h.logger.Debug("created jwt tokens secret in kubernetes", "uid", secret.UID)
	}

	accessToken, hasToken := secret.Data["accessToken"]
	if !hasToken {
		return errors.New("jwt tokens secret has no access token in data")
	}

	refreshToken, hasToken := secret.Data["refreshToken"]
	if !hasToken {
		return errors.New("jwt tokens secret has no refresh token in data")
	}

	h.jwtAccessTokenSecret = string(accessToken)
	h.jwtRefreshTokenSecret = string(refreshToken)

	return nil
}
