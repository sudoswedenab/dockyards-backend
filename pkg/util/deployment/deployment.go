package deployment

import (
	"path"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/containers/image/v5/docker/reference"
)

func AddNormalizedName(deployment *v1.Deployment) error {
	if deployment.ContainerImage != nil {
		named, err := reference.ParseNormalizedNamed(*deployment.ContainerImage)
		if err != nil {
			return err
		}

		deployment.ContainerImage = util.Ptr(named.String())

		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			base := path.Base(named.Name())
			deployment.Name = &base
		}

		deployment.Type = v1.DeploymentTypeContainerImage
	}

	if deployment.HelmChart != nil {
		if deployment.Name == nil || (deployment.Name != nil && *deployment.Name == "") {
			deployment.Name = deployment.HelmChart
		}

		deployment.Type = v1.DeploymentTypeHelm
	}

	if deployment.Kustomize != nil {
		if deployment.Name == nil {
			deployment.Name = util.Ptr("")
		}

		deployment.Type = v1.DeploymentTypeKustomize
	}

	if deployment.Namespace == nil {
		deployment.Namespace = deployment.Name
	}

	return nil
}
