package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) CreateNodePool(clusterOptions *model.ClusterOptions, clusterID string) (*model.NodePool, error) {
	customNodeTemplate, err := r.clusterOptionsToNodeTemplate(clusterOptions)
	if err != nil {
		return nil, err
	}

	var createdNodeTemplate CustomNodeTemplate
	err = r.ManagementClient.APIBaseClient.Create(managementv3.NodeTemplateType, &customNodeTemplate, &createdNodeTemplate)
	if err != nil {
		return nil, err
	}

	// Create Nodetemplate
	// + tester, använd samma typ av logik som finns i rancher configs
	//Call create node template
	opts := managementv3.NodePool{
		ClusterID:               clusterID,
		ControlPlane:            true,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    true,
		HostnamePrefix:          clusterOptions.Name + "-node-",
		Name:                    "",
		NamespaceId:             "",
		NodeTaints:              []managementv3.Taint{},
		NodeTemplateID:          createdNodeTemplate.ID,
		Quantity:                3,
		Worker:                  true,
	}

	createdNodePool, err := r.ManagementClient.NodePool.Create(&opts)
	if err != nil {
		return nil, err
	}

	nodePool := model.NodePool{
		Name: createdNodePool.Name,
	}

	return &nodePool, nil
}
