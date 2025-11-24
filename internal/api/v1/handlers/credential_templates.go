// Copyright 2025 Sudo Sweden AB
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
	"github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) ListCredentialTemplates(ctx context.Context) (*[]types.CredentialTemplate, error) {
	publicNamespace := h.DockyardsConfig.GetConfigKey(config.KeyPublicNamespace, "dockyards-public")

	var credentialTemplates dockyardsv1.CredentialTemplateList
	err := h.List(ctx, &credentialTemplates, client.InNamespace(publicNamespace))
	if err != nil {
		return nil, err
	}

	templates := make([]types.CredentialTemplate, len(credentialTemplates.Items))[:0]
	for _, template := range credentialTemplates.Items {
		var options *[]types.CredentialOption
		if template.Spec.Options != nil {
			items := make([]types.CredentialOption, len(template.Spec.Options))[:0]
			for _, option := range template.Spec.Options {
				var value types.CredentialOption
				value.Key = option.Key
				if option.Default != "" {
					value.Default = &option.Default
				}
				if option.DisplayName != "" {
					value.DisplayName = &option.DisplayName
				}
				if option.Plaintext {
					value.Plaintext = &option.Plaintext
				}
				if option.Type != "" {
					value.Type = &option.Type
				}
				items = append(items, value)
			}
			options = &items
		}

		templates = append(templates, types.CredentialTemplate{
			Name: template.Name,
			Options: options,
		})
	}

	return &templates, nil
}
