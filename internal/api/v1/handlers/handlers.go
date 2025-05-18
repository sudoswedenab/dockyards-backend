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
	contentJSON := middleware.NewContentType("application/json").Handler
	contentYAML := middleware.NewContentType("application/yaml").Handler

	validateJSON, err := middleware.NewValidateJSON()
	if err != nil {
		return err
	}

	mux.Handle("POST /v1/login",
		logger(
			validateJSON.WithSchema("#login")(http.HandlerFunc(h.Login)),
		),
	)

	mux.Handle("GET /v1/identity-providers", logger(http.HandlerFunc(h.ListIdentityProviders)))

	mux.Handle("POST /v1/refresh", logger(http.HandlerFunc(h.PostRefresh)))

	mux.Handle("GET /v1/cluster-options", logger(requireAuth(http.HandlerFunc(h.GetClusterOptions))))

	mux.Handle("GET /v1/orgs", logger(requireAuth(contentJSON(ListGlobalResource("organizations", h.ListGlobalOrganizations)))))
	mux.Handle("POST /v1/orgs", logger(requireAuth(contentJSON(CreateGlobalResource("organizations", h.CreateGlobalOrganization)))))
	mux.Handle("DELETE /v1/orgs/{resourceName}", logger(requireAuth(DeleteGlobalResource(&h, "organizations", h.DeleteGlobalOrganization))))
	mux.Handle("GET /v1/orgs/{resourceName}", logger(requireAuth(GetGlobalResource(&h, "organizations", h.GetGlobalOrganization))))

	mux.Handle("PATCH /v1/orgs/{resourceName}",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#updateOrganization")(UpdateGlobalResource(&h, "organizations", h.UpdateGlobalOrganization)),
				),
			),
		),
	)

	mux.Handle("POST /v1/orgs/{organizationName}/clusters",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#clusterOptions")(CreateOrganizationResource(&h, "clusters", h.CreateOrganizationCluster)),
				),
			),
		),
	)

	mux.Handle("GET /v1/whoami", logger(requireAuth(contentJSON(http.HandlerFunc(h.GetWhoami)))))

	mux.Handle("POST /v1/orgs/{organizationName}/credentials",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#createCredential")(CreateOrganizationResource(&h, "clusters", h.CreateOrganizationCredential)),
				),
			),
		),
	)

	mux.Handle("DELETE /v1/orgs/{organizationName}/credentials/{resourceName}", logger(requireAuth(DeleteOrganizationResource(&h, "clusters", h.DeleteOrganizationCredential))))
	mux.Handle("GET /v1/orgs/{organizationName}/credentials", logger(requireAuth(contentJSON(ListOrganizationResource(&h, "clusters", h.ListOrganizationCredentials)))))
	mux.Handle("GET /v1/orgs/{organizationName}/credentials/{resourceName}", logger(requireAuth(contentJSON(GetOrganizationResource(&h, "clusters", h.GetOrganizationCredential)))))

	mux.Handle("PATCH /v1/orgs/{organizationName}/credentials/{resourceName}",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#updateCredential")(UpdateOrganizationResource(&h, "clusters", h.UpdateOrganizationCredential)),
				),
			),
		),
	)

	mux.Handle("POST /v1/orgs/{organizationName}/clusters/{clusterName}/workloads",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#workloadOptions")(CreateClusterResource(&h, "workloads", h.CreateClusterWorkload)),
				),
			),
		),
	)

	mux.Handle("DELETE /v1/orgs/{organizationName}/clusters/{clusterName}/workloads/{resourceName}", logger(requireAuth(DeleteClusterResource(&h, "workloads", h.DeleteClusterWorkload))))

	mux.Handle("PUT /v1/orgs/{organizationName}/clusters/{clusterName}/workloads/{resourceName}",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#workloadOptions")(UpdateClusterResource(&h, "workloads", h.UpdateClusterWorkload)),
				),
			),
		),
	)

	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{clusterName}/workloads", logger(requireAuth(contentJSON(ListClusterResource(&h, "workloads", h.ListClusterWorkloads)))))
	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{clusterName}/workloads/{resourceName}", logger(requireAuth(contentJSON(GetClusterResource(&h, "workloads", h.GetClusterWorkload)))))

	mux.Handle("POST /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#nodePoolOptions")(CreateClusterResource(&h, "nodepools", h.CreateClusterNodePool)),
				),
			),
		),
	)

	mux.Handle("DELETE /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools/{resourceName}", logger(requireAuth(DeleteClusterResource(&h, "nodepools", h.DeleteClusterNodePool))))
	mux.Handle("DELETE /v1/orgs/{organizationName}/clusters/{resourceName}", logger(requireAuth(DeleteOrganizationResource(&h, "clusters", h.DeleteOrganizationCluster))))
	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools/{resourceName}", logger(requireAuth(contentJSON(GetClusterResource(&h, "nodepools", h.GetClusterNodePool)))))
	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools", logger(requireAuth(contentJSON(ListClusterResource(&h, "nodepools", h.ListClusterNodePools)))))
	mux.Handle("PATCH /v1/orgs/{organizationName}/clusters/{clusterName}/node-pools/{resourceName}", logger(requireAuth(UpdateClusterResource(&h, "nodepools", h.UpdateClusterNodePool))))

	mux.Handle("GET /v1/orgs/{organizationName}/clusters", logger(requireAuth(contentJSON(ListOrganizationResource(&h, "clusters", h.ListOrganizationClusters)))))
	mux.Handle("GET /v1/orgs/{organizationName}/clusters/{resourceName}", logger(requireAuth(contentJSON(GetOrganizationResource(&h, "clusters", h.GetOrganizationCluster)))))

	mux.Handle("POST /v1/orgs/{organizationName}/clusters/{clusterName}/kubeconfig", logger(requireAuth(contentYAML(CreateClusterResource(&h, "clusters", h.CreateClusterKubeconfig)))))

	mux.Handle("POST /v1/orgs/{organizationName}/invitations",
		logger(
			requireAuth(
				contentJSON(
					validateJSON.WithSchema("#createInvitation")(CreateOrganizationResource(&h, "invitations", h.CreateOrganizationInvitation)),
				),
			),
		),
	)

	mux.Handle("DELETE /v1/orgs/{organizationName}/invitations/{resourceName}", logger(requireAuth(DeleteOrganizationResource(&h, "invitations", h.DeleteOrganizationInvitation))))
	mux.Handle("GET /v1/orgs/{organizationName}/invitations", logger(requireAuth(contentJSON(ListOrganizationResource(&h, "invitations", h.ListOrganizationInvitations)))))

	mux.Handle("GET /v1/invitations", logger(requireAuth(contentJSON(ListGlobalResource("invitations", h.ListGlobalInvitations)))))

	return nil
}
