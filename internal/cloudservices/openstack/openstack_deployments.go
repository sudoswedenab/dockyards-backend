package openstack

import (
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
)

func (s *openStackService) GetClusterDeployments(organization *model.Organization, cluster *model.Cluster) (*[]model.Deployment, error) {
	openStackOrganization, err := s.getOpenStackOrganization(organization)
	if err != nil {
		s.logger.Error("error getting openstack organization", "name", organization.Name, "err", err)
		return nil, err
	}

	cloudConf := []string{
		"[Global]",
		"auth-url=" + s.authOptions.IdentityEndpoint,
		"application-credential-id=" + openStackOrganization.ApplicationCredentialID,
		"application-credential-secret=" + openStackOrganization.ApplicationCredentialSecret,
	}

	openStackCinderCSIDeployment := model.Deployment{
		ClusterID:      cluster.ID,
		Name:           "openstack-cinder-csi",
		HelmChart:      "openstack-cinder-csi",
		HelmRepository: "https://kubernetes.github.io/cloud-provider-openstack",
		HelmVersion:    "2.28.0-alpha.3",
		Namespace:      "kube-system",
		HelmValues: model.HelmValues{
			"secret": map[string]any{
				"enabled":  true,
				"create":   true,
				"filename": "cloud.conf",
				"name":     "cinder-csi-cloud-config",
				"data": map[string]interface{}{
					"cloud.conf": strings.Join(cloudConf, "\n"),
				},
			},
			"storageClass": map[string]any{
				"delete": map[string]any{
					"isDefault": true,
				},
			},
			"clusterID": cluster.Name,
		},
	}

	clusterApps := []model.Deployment{
		openStackCinderCSIDeployment,
	}

	return &clusterApps, nil
}
