package openstack

import (
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
)

func (s *openStackService) GetClusterDeployments(organization *v1.Organization, cluster *v1.Cluster) (*[]v1.Deployment, error) {
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

	openStackCinderCSIDeployment := v1.Deployment{
		ClusterID:      cluster.ID,
		Name:           util.Ptr("openstack-cinder-csi"),
		HelmChart:      util.Ptr("openstack-cinder-csi"),
		HelmRepository: util.Ptr("https://kubernetes.github.io/cloud-provider-openstack"),
		HelmVersion:    util.Ptr("2.28.0"),
		Namespace:      util.Ptr("kube-system"),
		HelmValues: util.Ptr(map[string]any{
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
		}),
	}

	clusterApps := []v1.Deployment{
		openStackCinderCSIDeployment,
	}

	return &clusterApps, nil
}
