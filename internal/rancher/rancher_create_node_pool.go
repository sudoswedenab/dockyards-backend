package rancher

import (
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) CreateNodePool(organization *model.Organization, cluster *model.Cluster, nodePoolOptions *model.NodePoolOptions) (*model.NodePool, error) {
	cloudConfig, err := r.cloudService.PrepareEnvironment(organization, cluster, nodePoolOptions)
	if err != nil {
		return nil, err
	}

	secGroups := strings.Join(cloudConfig.SecurityGroups, ",")

	openstackConfig := openstackConfig{
		AuthURL:                     cloudConfig.AuthURL,
		ApplicationCredentialID:     cloudConfig.ApplicationCredentialID,
		ApplicationCredentialSecret: cloudConfig.ApplicationCredentialSecret,
		FlavorID:                    cloudConfig.FlavorID,
		ImageID:                     cloudConfig.ImageID,
		KeypairName:                 cloudConfig.KeypairName,
		NetID:                       cloudConfig.NetID,
		PrivateKeyFile:              cloudConfig.PrivateKeyFile,
		SecGroups:                   secGroups,
		SSHUser:                     "ubuntu",
	}

	nodeTemplateName := cluster.Name + "-" + nodePoolOptions.Name
	customNodeTemplate := CustomNodeTemplate{
		NodeTemplate: managementv3.NodeTemplate{
			Name: nodeTemplateName,
		},
		OpenstackConfig: &openstackConfig,
	}

	var createdNodeTemplate CustomNodeTemplate
	err = r.managementClient.APIBaseClient.Create(managementv3.NodeTemplateType, &customNodeTemplate, &createdNodeTemplate)
	if err != nil {
		return nil, err
	}

	encodedName := encodeName(cluster.Organization, cluster.Name)

	hostnamePrefix := encodedName + "-" + nodePoolOptions.Name + "-"
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

	createdNodePool, err := r.managementClient.NodePool.Create(&opts)
	if err != nil {
		return nil, err
	}

	nodePool := model.NodePool{
		Name: createdNodePool.Name,
	}

	return &nodePool, nil
}
