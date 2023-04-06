package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) CreateNodePool(cluster *model.Cluster, nodePoolOptions *model.NodePoolOptions) (*model.NodePool, error) {
	openstackConfig, err := r.prepareOpenstackEnvironment(cluster, nodePoolOptions)
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
	hostnamePrefix := cluster.Name + "-" + nodePoolOptions.Name + "-"
	opts := managementv3.NodePool{
		ClusterID:               cluster.ID,
		ControlPlane:            nodePoolOptions.ControlPlane,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    nodePoolOptions.Etcd,
		HostnamePrefix:          hostnamePrefix,
		Name:                    nodePoolOptions.Name,
		NamespaceId:             "",
		NodeTaints:              []managementv3.Taint{},
		NodeTemplateID:          createdNodeTemplate.ID,
		Quantity:                int64(nodePoolOptions.Quantity),
		Worker:                  !nodePoolOptions.ControlPlaneComponentsOnly,
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
