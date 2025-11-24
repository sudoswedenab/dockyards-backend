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
	"context"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/config"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=features,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

func (h *handler) GetClusterOptions(ctx context.Context) (*types.Options, error) {
	publicNamespace := h.DockyardsConfig.GetConfigKey(config.KeyPublicNamespace, "dockyards-public")

	release, err := apiutil.GetDefaultRelease(ctx, h.Client, dockyardsv1.ReleaseTypeKubernetes)
	if err != nil {
		return nil, err
	}

	response := types.Options{
		Version: []string{},
	}

	if release != nil {
		response.Version = release.Status.Versions
	}

	storageRoleFeatureEnabled, err := apiutil.IsFeatureEnabled(ctx, h.Client, featurenames.FeatureStorageRole, publicNamespace)
	if err != nil {
		return nil, err
	}

	if storageRoleFeatureEnabled {
		storageResourceTypes := []string{}

		hostPathFeatureEnabled, err := apiutil.IsFeatureEnabled(ctx, h.Client, featurenames.FeatureStorageResourceTypeHostPath, publicNamespace)
		if err != nil {
			return nil, err
		}

		if hostPathFeatureEnabled {
			storageResourceTypes = append(storageResourceTypes, dockyardsv1.StorageResourceTypeHostPath)
		}

		response.StorageResourceTypes = &storageResourceTypes
	}

	return &response, nil
}
