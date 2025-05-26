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

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

func (h *handler) GetClusterOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	release, err := apiutil.GetDefaultRelease(ctx, h.Client, dockyardsv1.ReleaseTypeKubernetes)
	if err != nil {
		logger.Error("error getting default release", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	options := types.Options{
		Version: []string{},
	}

	if release != nil {
		options.Version = release.Status.Versions
	}

	storageRoleFeatureEnabled, err := apiutil.IsFeatureEnabled(ctx, h.Client, featurenames.FeatureStorageRole, h.namespace)
	if err != nil {
		logger.Error("error verifying feature", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if storageRoleFeatureEnabled {
		storageResourceTypes := []string{}

		hostPathFeatureEnabled, err := apiutil.IsFeatureEnabled(ctx, h.Client, featurenames.FeatureStorageResourceTypeHostPath, h.namespace)
		if err != nil {
			logger.Error("error verifying feature", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if hostPathFeatureEnabled {
			storageResourceTypes = append(storageResourceTypes, dockyardsv1.StorageResourceTypeHostPath)
		}

		options.StorageResourceTypes = &storageResourceTypes
	}

	b, err := json.Marshal(&options)
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
