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
