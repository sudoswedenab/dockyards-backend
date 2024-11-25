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

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetOrgs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Debug("error fetching user from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	matchingFields := client.MatchingFields{
		index.MemberReferencesField: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error listing organizations in kubernetes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	var organizations []types.Organization
	for _, organization := range organizationList.Items {
		v1Organization := types.Organization{
			ID:   string(organization.UID),
			Name: organization.Name,
		}

		organizations = append(organizations, v1Organization)
	}

	b, err := json.Marshal(&organizations)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}
