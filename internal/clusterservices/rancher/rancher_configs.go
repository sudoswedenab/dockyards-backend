package rancher

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) clusterOptionsToNodeTemplate(clusterOptions *v1.ClusterOptions, config *openstackConfig) (*CustomNodeTemplate, error) {
	customNodeTemplate := CustomNodeTemplate{
		NodeTemplate: managementv3.NodeTemplate{
			Name: clusterOptions.Name,
		},
		OpenstackConfig: config,
	}

	return &customNodeTemplate, nil
}

func (r *rancher) GetSupportedVersions() ([]string, error) {
	setting, err := r.managementClient.Setting.ByID("k8s-versions-current")
	if err != nil {
		return []string{}, err
	}

	versions := strings.Split(setting.Value, ",")

	slices.Sort(versions)
	slices.Reverse(versions)

	return versions, nil
}
