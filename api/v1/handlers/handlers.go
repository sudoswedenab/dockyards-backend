package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type handler struct {
	db                    *gorm.DB
	clusterService        clusterservices.ClusterService
	logger                *slog.Logger
	jwtAccessTokenSecret  string
	jwtRefreshTokenSecret string
	cloudService          cloudservices.CloudService
	gitProjectRoot        string
	controllerClient      client.Client
	namespace             string
}

type HandlerOption func(*handler)

func WithCloudService(cloudService cloudservices.CloudService) HandlerOption {
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

func WithClusterService(clusterService clusterservices.ClusterService) HandlerOption {
	return func(h *handler) {
		h.clusterService = clusterService
	}
}

func WithGitProjectRoot(gitProjectRoot string) HandlerOption {
	return func(h *handler) {
		h.gitProjectRoot = gitProjectRoot
	}
}

func WithManager(manager ctrl.Manager) HandlerOption {
	controllerClient := manager.GetClient()
	return func(h *handler) {
		h.controllerClient = controllerClient
	}
}

func WithNamespace(namespace string) HandlerOption {
	return func(h *handler) {
		h.namespace = namespace
	}
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *slog.Logger, handlerOptions ...HandlerOption) error {
	methodNotAllowed := func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
	}

	h := handler{
		db:     db,
		logger: logger,
	}

	for _, handlerOption := range handlerOptions {
		handlerOption(&h)
	}

	if h.jwtAccessTokenSecret == "" || h.jwtRefreshTokenSecret == "" {
		logger.Warn("using empty jwt tokens")
	}

	if h.namespace == "" {
		logger.Warn("using empty namespace")
	}

	middlewareHandler := middleware.Handler{
		DB:                 db,
		Logger:             logger,
		AccessTokenSecret:  h.jwtAccessTokenSecret,
		RefreshTokenSecret: h.jwtRefreshTokenSecret,
	}

	if h.gitProjectRoot == "" {
		logger.Warn("no git project root set, using '/var/www/git'")

		h.gitProjectRoot = "/var/www/git"
	}

	r.POST("/v1/login", h.Login)
	r.POST("/v1/refresh", h.PostRefresh)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
	g.POST("/logout", h.Logout)
	g.GET("/cluster-options", h.GetClusterOptions)

	g.GET("/clusters", h.GetClusters)
	g.GET("/clusters/:clusterID", h.GetCluster)
	g.DELETE("/clusters/:clusterID", h.DeleteCluster)

	g.GET("/orgs", h.GetOrgs)
	g.PUT("/orgs", methodNotAllowed)
	g.DELETE("/orgs", methodNotAllowed)

	g.POST("/orgs/:org/clusters", h.PostOrgClusters)

	g.GET("/clusters/:clusterID/deployments", h.GetClusterDeployments)
	g.POST("/clusters/:clusterID/deployments", h.PostClusterDeployments)
	g.GET("/clusters/:clusterID/kubeconfig", h.GetClusterKubeconfig)
	g.POST("/clusters/:clusterID/node-pools", h.PostClusterNodePools)

	g.GET("/deployments/:deploymentID", h.GetDeployment)
	g.DELETE("/deployments/:deploymentID", h.DeleteDeployment)

	g.GET("/credentials", h.GetCredentials)
	g.GET("/credentials/:uuid", h.GetCredentialUUID)
	g.POST("/orgs/:org/credentials", h.PostOrgCredentials)
	g.DELETE("orgs/:org/credentials/:credential", h.DeleteOrgCredentials)

	g.GET("/overview", h.GetOverview)
	g.GET("/whoami", h.GetWhoami)

	g.GET("/apps", h.GetApps)
	g.GET("/apps/:appID", h.GetApp)

	g.GET("/node-pools/:nodePoolID", h.GetNodePool)
	g.DELETE("/node-pools/:nodePoolID", h.DeleteNodePool)

	return nil
}

func (h *handler) getSubjectFromContext(c *gin.Context) (string, error) {
	v, exists := c.Get("sub")
	if !exists {
		return "", errors.New("error fecthing subject from context")
	}

	sub, ok := v.(string)
	if !ok {
		return "", errors.New("error during type conversion")
	}

	return sub, nil
}

func (h *handler) isMember(subject string, organization *v1alpha1.Organization) bool {
	for _, memberRef := range organization.Spec.MemberRefs {
		if string(memberRef.UID) == subject && memberRef.Kind == v1alpha1.UserKind {
			return true
		}
	}

	return false
}
