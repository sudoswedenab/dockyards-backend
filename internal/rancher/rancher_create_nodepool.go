package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) CreateNodePool(cluster *model.Cluster) (*model.NodePool, error) {
	nodePoolOptions := model.NodePoolOptions{
		Name: cluster.Name,
	}
	openstackConfig, err := r.prepareOpenstackEnvironment(cluster.Name, nodePoolOptions)
	if err != nil {
		return nil, err
	}

	customNodeTemplate := CustomNodeTemplate{
		NodeTemplate: managementv3.NodeTemplate{
			Name: cluster.Name,
		},
		OpenstackConfig: openstackConfig,
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
		ClusterID:               cluster.ID,
		ControlPlane:            true,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    true,
		HostnamePrefix:          cluster.Name + "-node-",
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
