// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"crypto/ecdsa"
	"log/slog"
	"net/http"
	"path/filepath"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
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

	validateJSON, err := middleware.NewValidateJSON(filepath.Join("internal", "api", "v1", "middleware"))
	if err != nil {
		return err
	}

	mux.Handle("POST /v1/login",
		logger(
			validateJSON.WithSchema("#login")(http.HandlerFunc(h.Login)),
		),
	)

	mux.Handle("POST /v1/refresh", logger(http.HandlerFunc(h.PostRefresh)))

	mux.Handle("GET /v1/cluster-options", logger(requireAuth(http.HandlerFunc(h.GetClusterOptions))))

	mux.Handle("GET /v1/clusters/{clusterID}", logger(requireAuth(http.HandlerFunc(h.GetCluster))))
	mux.Handle("GET /v1/clusters/{clusterID}/kubeconfig", logger(requireAuth(http.HandlerFunc(h.GetClusterKubeconfig))))
	mux.Handle("POST /v1/clusters/{clusterID}/node-pools", logger(requireAuth(http.HandlerFunc(h.PostClusterNodePools))))

	mux.Handle("GET /v1/orgs", logger(requireAuth(http.HandlerFunc(h.GetOrgs))))
	mux.Handle("POST /v1/orgs/{organizationName}/clusters",
		logger(
			requireAuth(
				validateJSON.WithSchema("#clusterOptions")(CreateOrganizationResource(&h, "clusters", h.CreateOrganizationCluster)),
			),
		),
	)

	mux.Handle("GET /v1/whoami", logger(requireAuth(http.HandlerFunc(h.GetWhoami))))

	mux.Handle("GET /v1/apps", logger(requireAuth(http.HandlerFunc(h.GetApps))))
	mux.Handle("GET /v1/apps/{appID}", logger(requireAuth(http.HandlerFunc(h.GetApp))))

	mux.Handle("GET /v1/node-pools/{nodePoolID}", logger(requireAuth(http.HandlerFunc(h.GetNodePool))))
	mux.Handle("PATCH /v1/node-pools/{nodePoolID}", logger(requireAuth(http.HandlerFunc(h.UpdateNodePool))))

	mux.Handle("DELETE /v1/orgs/{organizationName}/credentials/{credentialName}", logger(requireAuth(DeleteOrganizationResource(&h, "clusters", h.DeleteOrganizationCredential))))
	mux.Handle("GET /v1/orgs/{organizationName}/credentials", logger(requireAuth(ListOrganizationResource(&h, "clusters", h.ListOrganizationCredentials))))
	mux.Handle("GET /v1/orgs/{organizationName}/credentials/{credentialName}", logger(requireAuth(http.HandlerFunc(h.GetOrganizationCredential))))
	mux.Handle("PUT /v1/orgs/{organizationName}/credentials/{credentialName}", logger(requireAuth(http.HandlerFunc(h.PutOrganizationCredential))))

	mux.Handle("GET /v1/credentials", logger(requireAuth(http.HandlerFunc(h.GetCredentials))))

	mux.Handle("POST /v1/orgs/{organizationName}/clusters/{clusterName}/workloads",
		logger(
			requireAuth(
				validateJSON.WithSchema("#workload")(CreateClusterResource(&h, "workloads", h.CreateClusterWorkload)),
			),
		),
	)

	mux.Handle("DELETE /v1/orgs/{organizationName}/clusters/{clusterName}/workloads/{resourceName}", logger(requireAuth(DeleteClusterResource(&h, "workloads", h.DeleteClusterWorkload))))

	mux.Handle("PUT /v1/orgs/{organizationName}/clusters/{clusterName}/workloads/{workloadName}",
		logger(
			requireAuth(
				validateJSON.WithSchema("#workload")(http.HandlerFunc(h.UpdateClusterWorkload)),
			),
		),
	)

	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{clusterName}/workloads", logger(requireAuth(ListClusterResource(&h, "workloads", h.ListClusterWorkloads))))
	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{clusterName}/workloads/{workloadName}", logger(requireAuth(http.HandlerFunc(h.GetClusterWorkload))))

	mux.Handle("POST /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools",
		logger(
			requireAuth(
				validateJSON.WithSchema("#nodePoolOptions")(CreateClusterResource(&h, "nodepools", h.CreateClusterNodePool)),
			),
		),
	)

	mux.Handle("POST /v1/orgs/{organizationName}/credentials",
		logger(
			requireAuth(
				validateJSON.WithSchema("#credential")(CreateOrganizationResource(&h, "clusters", h.CreateOrganizationCredential)),
			),
		),
	)

	mux.Handle("DELETE /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools/{resourceName}", logger(requireAuth(DeleteClusterResource(&h, "nodepools", h.DeleteClusterNodePool))))
	mux.Handle("DELETE /v1/orgs/{organizationName}/clusters/{clusterName}", logger(requireAuth(DeleteOrganizationResource(&h, "clusters", h.DeleteOrganizationCluster))))

	mux.Handle("GET /v1/orgs/{organizationName}/clusters", logger(requireAuth(ListOrganizationResource(&h, "clusters", h.ListOrganizationClusters))))

	return nil
}
