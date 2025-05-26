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
	"encoding/json"
	"net/http"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
)

func (h *handler) ListIdentityProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	var idplist dockyardsv1.IdentityProviderList
	if err := h.List(ctx, &idplist); err != nil {
		logger.Error("missing resource", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
	idps := []types.IdentityProvider{}
	for _, idp := range idplist.Items {
		// Only return objects with some type of config (currently just OIDC)
		if idp.Spec.OIDCConfig == nil {
			logger.Warn("incomplete IdentityProvider", "name", idp.Name)

			continue
		}
		// Only return OIDC objects which have at least one way of configuring an OIDC provider
		if idp.Spec.OIDCConfig != nil && idp.Spec.OIDCConfig.OIDCProviderDiscoveryURL == nil && idp.Spec.OIDCConfig.OIDCProviderConfig == nil {
			logger.Warn("incomplete or misconfigured OIDCConfig", "name", idp.Name)

			continue
		}
		idps = append(idps, types.IdentityProvider{
			ID:          string(idp.GetUID()),
			Name:        idp.GetName(),
			DisplayName: idp.Spec.DisplayName,
		})
	}

	b, err := json.Marshal(&idps)
	if err != nil {
		logger.Error("error serializing identity providers", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}
