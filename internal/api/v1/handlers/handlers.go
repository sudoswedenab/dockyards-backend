package handlers

import (
	"crypto/ecdsa"
	"log/slog"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type handler struct {
	client.Client

	logger               *slog.Logger
	namespace            string
	jwtAccessPrivateKey  *ecdsa.PrivateKey
	jwtRefreshPrivateKey *ecdsa.PrivateKey
	jwtAccessPublicKey   *ecdsa.PublicKey
	jwtRefreshPublicKey  *ecdsa.PublicKey
}

type HandlerOption func(*handler)

func WithManager(mgr ctrl.Manager) HandlerOption {
	controllerClient := mgr.GetClient()

	return func(h *handler) {
		h.Client = controllerClient
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

func WithLogger(logger *slog.Logger) HandlerOption {
	return func(h *handler) {
		h.logger = logger
	}
}

func RegisterRoutes(mux *http.ServeMux, handlerOptions ...HandlerOption) error {
	var h handler

	for _, handlerOption := range handlerOptions {
		handlerOption(&h)
	}

	if h.namespace == "" {
		h.logger.Warn("using empty namespace")
	}

	logger := middleware.NewLogger(h.logger).Handler
	requireAuth := middleware.NewRequireAuth(h.jwtAccessPublicKey).Handler

	mux.Handle("POST /v1/login", logger(http.HandlerFunc(h.Login)))
	mux.Handle("POST /v1/refresh", logger(http.HandlerFunc(h.PostRefresh)))

	mux.Handle("GET /v1/cluster-options", logger(requireAuth(http.HandlerFunc(h.GetClusterOptions))))

	mux.Handle("DELETE /v1/clusters/{clusterID}", logger(requireAuth(http.HandlerFunc(h.DeleteCluster))))
	mux.Handle("GET /v1/clusters", logger(requireAuth(http.HandlerFunc(h.GetClusters))))
	mux.Handle("GET /v1/clusters/{clusterID}", logger(requireAuth(http.HandlerFunc(h.GetCluster))))
	mux.Handle("GET /v1/clusters/{clusterID}/deployments", logger(requireAuth(http.HandlerFunc(h.GetClusterDeployments))))
	mux.Handle("GET /v1/clusters/{clusterID}/kubeconfig", logger(requireAuth(http.HandlerFunc(h.GetClusterKubeconfig))))
	mux.Handle("POST /v1/clusters/{clusterID}/deployments", logger(requireAuth(http.HandlerFunc(h.PostClusterDeployments))))
	mux.Handle("POST /v1/clusters/{clusterID}/node-pools", logger(requireAuth(http.HandlerFunc(h.PostClusterNodePools))))

	mux.Handle("GET /v1/orgs", logger(requireAuth(http.HandlerFunc(h.GetOrgs))))
	mux.Handle("GET /v1/orgs/{organizationID}/credentials", logger(requireAuth(http.HandlerFunc(h.GetOrgCredentials))))
	mux.Handle("POST /v1/orgs/{organizationID}/clusters", logger(requireAuth(http.HandlerFunc(h.PostOrgClusters))))
	mux.Handle("POST /v1/orgs/{organizationID}/credentials", logger(requireAuth(http.HandlerFunc(h.PostOrgCredentials))))

	mux.Handle("GET /v1/deployments/{deploymentID}", logger(requireAuth(http.HandlerFunc(h.GetDeployment))))

	mux.Handle("GET /v1/credentials/{credentialID}", logger(requireAuth(http.HandlerFunc(h.GetCredential))))
	mux.Handle("DELETE /v1/credentials/{credentialID}", logger(requireAuth(http.HandlerFunc(h.DeleteCredential))))

	mux.Handle("GET /v1/whoami", logger(requireAuth(http.HandlerFunc(h.GetWhoami))))

	mux.Handle("GET /v1/apps", logger(requireAuth(http.HandlerFunc(h.GetApps))))
	mux.Handle("GET /v1/apps/{appID}", logger(requireAuth(http.HandlerFunc(h.GetApp))))

	mux.Handle("GET /v1/node-pools/{nodePoolID}", logger(requireAuth(http.HandlerFunc(h.GetNodePool))))
	mux.Handle("DELETE /v1/node-pools/{nodePoolID}", logger(requireAuth(http.HandlerFunc(h.DeleteNodePool))))

	mux.Handle("DELETE /v1/deployments/{deploymentID}", logger(requireAuth(http.HandlerFunc(h.DeleteDeployment))))

	return nil
}

func (h *handler) isMember(subject string, organization *dockyardsv1.Organization) bool {
	for _, memberRef := range organization.Spec.MemberRefs {
		if memberRef.UID == types.UID(subject) {
			return true
		}
	}

	return false
}
