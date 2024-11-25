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

package deployment

import (
	"path"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"github.com/containers/image/v5/docker/reference"
	"k8s.io/utils/ptr"
)

func AddNormalizedName(deployment *types.Deployment) error {
	if deployment.ContainerImage != nil {
		named, err := reference.ParseNormalizedNamed(*deployment.ContainerImage)
		if err != nil {
			return err
		}

		deployment.ContainerImage = ptr.To(named.String())

		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			base := path.Base(named.Name())
			deployment.Name = &base
		}

		deployment.Type = types.DeploymentTypeContainerImage
	}

	if deployment.HelmChart != nil {
		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			deployment.Name = deployment.HelmChart
		}

		deployment.Type = types.DeploymentTypeHelm
	}

	if deployment.Kustomize != nil {
		if deployment.Name == nil {
			deployment.Name = ptr.To("")
		}

		deployment.Type = types.DeploymentTypeKustomize
	}

	if deployment.Namespace == nil {
		deployment.Namespace = deployment.Name
	}

	return nil
}
