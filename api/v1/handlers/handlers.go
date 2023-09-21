package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultDockyardsNamespace = "dockyards"
	defaultJWTSecretName      = "jwt-tokens"
)

type handler struct {
	db                    *gorm.DB
	clusterService        clusterservices.ClusterService
	accessTokenName       string
	refreshTokenName      string
	logger                *slog.Logger
	jwtAccessTokenSecret  string
	jwtRefreshTokenSecret string
	cloudService          cloudservices.CloudService
	gitProjectRoot        string
	controllerClient      client.Client
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

func WithControllerClient(controllerClient client.Client) HandlerOption {
	return func(h *handler) {
		h.controllerClient = controllerClient
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

	if h.gitProjectRoot == "" {
		logger.Warn("no git project root set, using '/var/www/git'")

		h.gitProjectRoot = "/var/www/git"
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)
	r.POST("/v1/refresh", h.PostRefresh)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
	g.POST("/logout", h.Logout)
	g.GET("/cluster-options", h.GetClusterOptions)

	g.GET("/clusters", h.GetClusters)
	g.GET("/clusters/:clusterID", h.GetCluster)
	g.DELETE("/clusters/:clusterID", h.DeleteCluster)

	g.GET("/orgs", h.GetOrgs)
	g.POST("/orgs", h.PostOrgs)
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

	g.DELETE("/orgs/:org", h.DeleteOrganization)

	g.GET("/apps", h.GetApps)
	g.GET("/apps/:appID", h.GetApp)

	g.GET("/node-pools/:nodePoolID", h.GetNodePool)
	g.DELETE("/node-pools/:nodePoolID", h.DeleteNodePool)

	return nil
}

func (h *handler) getUserFromContext(c *gin.Context) (v1.User, error) {
	u, exists := c.Get("user")
	if !exists {
		return v1.User{}, errors.New("error fecthing user from context")
	}

	user, ok := u.(v1.User)
	if !ok {
		return v1.User{}, errors.New("error during user type conversion")
	}

	return user, nil
}

func (h *handler) isMember(user *v1.User, organization *v1.Organization) (bool, error) {
	var userOrganizations []v1.Organization
	err := h.db.Model(&user).Association("Organizations").Find(&userOrganizations)
	if err != nil {
		return false, err
	}

	isMember := false
	for _, userOrganization := range userOrganizations {
		if userOrganization.ID == organization.ID {
			isMember = true
		}
	}

	return isMember, nil

}

func (h *handler) setOrGenerateTokens() error {
	ctx := context.Background()

	objectKey := client.ObjectKey{
		Namespace: defaultDockyardsNamespace,
		Name:      defaultJWTSecretName,
	}

	var secret corev1.Secret
	err := h.controllerClient.Get(ctx, objectKey, &secret)
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

		err = h.controllerClient.Create(ctx, &secret)
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
