package handlers

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log/slog"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type handler struct {
	logger               *slog.Logger
	controllerClient     client.Client
	namespace            string
	jwtAccessPrivateKey  *ecdsa.PrivateKey
	jwtRefreshPrivateKey *ecdsa.PrivateKey
	jwtAccessPublicKey   *ecdsa.PublicKey
	jwtRefreshPublicKey  *ecdsa.PublicKey
}

type HandlerOption func(*handler)

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

func WithJWTPrivateKeys(accessKey, refreshKey *ecdsa.PrivateKey) HandlerOption {
	accessPublicKey := accessKey.PublicKey
	refreshPublicKey := refreshKey.PublicKey

	return func(h *handler) {
		h.jwtAccessPrivateKey = accessKey
		h.jwtRefreshPrivateKey = refreshKey
		h.jwtAccessPublicKey = &accessPublicKey
		h.jwtRefreshPublicKey = &refreshPublicKey
	}
}

func RegisterRoutes(r *gin.Engine, logger *slog.Logger, handlerOptions ...HandlerOption) error {
	methodNotAllowed := func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
	}

	h := handler{
		logger: logger,
	}

	for _, handlerOption := range handlerOptions {
		handlerOption(&h)
	}

	if h.namespace == "" {
		logger.Warn("using empty namespace")
	}

	middlewareHandler := middleware.Handler{
		Logger:          logger,
		AccessPublicKey: h.jwtAccessPublicKey,
	}

	r.POST("/v1/login", h.Login)
	r.POST("/v1/refresh", h.PostRefresh)

	g := r.Group("/v1", middlewareHandler.RequireAuth)
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

	g.GET("/orgs/:org/credentials", h.GetOrgCredentials)
	g.POST("/orgs/:org/credentials", h.PostOrgCredentials)

	g.GET("/credentials/:credentialID", h.GetCredential)
	g.DELETE("credentials/:credentialID", h.DeleteCredential)

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
		if memberRef.UID == types.UID(subject) {
			return true
		}
	}

	return false
}

func (h *handler) getOwnerOrganization(ctx context.Context, object client.Object) (*v1alpha1.Organization, error) {
	for _, ownerReference := range object.GetOwnerReferences() {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != v1alpha1.OrganizationKind {
			continue
		}

		var organization v1alpha1.Organization
		err := h.controllerClient.Get(ctx, client.ObjectKey{Name: ownerReference.Name}, &organization)
		if err != nil {
			return nil, err
		}

		return &organization, nil
	}

	return nil, nil
}

func (h *handler) getOwnerCluster(ctx context.Context, object client.Object) (*v1alpha1.Cluster, error) {
	for _, ownerReference := range object.GetOwnerReferences() {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != v1alpha1.ClusterKind {
			continue
		}

		var cluster v1alpha1.Cluster
		err := h.controllerClient.Get(ctx, client.ObjectKey{Name: ownerReference.Name, Namespace: object.GetNamespace()}, &cluster)
		if err != nil {
			return nil, err
		}

		return &cluster, nil
	}

	return nil, nil
}
