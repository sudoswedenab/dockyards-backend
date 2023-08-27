package rancher

import (
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	corev1 "k8s.io/api/core/v1"
)

func (r *rancher) CreateNodePool(organization *v1.Organization, cluster *v1.Cluster, nodePoolOptions *v1.NodePoolOptions) (*v1.NodePool, error) {
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

	nodeTaints := []managementv3.Taint{}
	nodeLabels := map[string]string{}

	if nodePoolOptions.LoadBalancer != nil && *nodePoolOptions.LoadBalancer {
		taint := managementv3.Taint{
			Effect: string(corev1.TaintEffectNoSchedule),
			Key:    TaintNodeRoleLoadBalancer,
		}
		nodeTaints = append(nodeTaints, taint)

		nodeLabels[LabelNodeRoleLoadBalancer] = ""
	}

	nodeTemplateName := cluster.Name + "-" + nodePoolOptions.Name
	customNodeTemplate := CustomNodeTemplate{
		NodeTemplate: managementv3.NodeTemplate{
			Name:   nodeTemplateName,
			Labels: nodeLabels,
		},
		OpenstackConfig: &openstackConfig,
	}

	var createdNodeTemplate CustomNodeTemplate
	err = r.managementClient.APIBaseClient.Create(managementv3.NodeTemplateType, &customNodeTemplate, &createdNodeTemplate)
	if err != nil {
		return nil, err
	}

	hostnamePrefix := cluster.Name + "-" + nodePoolOptions.Name + "-"
	opts := managementv3.NodePool{
		ClusterID:               cluster.ID,
		ControlPlane:            *nodePoolOptions.ControlPlane,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    *nodePoolOptions.Etcd,
		HostnamePrefix:          hostnamePrefix,
		Name:                    nodePoolOptions.Name,
		NamespaceId:             "",
		NodeTaints:              nodeTaints,
		NodeTemplateID:          createdNodeTemplate.ID,
		Quantity:                int64(*nodePoolOptions.Quantity),
		Worker:                  !*nodePoolOptions.ControlPlaneComponentsOnly,
	}

	createdNodePool, err := r.managementClient.NodePool.Create(&opts)
	if err != nil {
		return nil, err
	}

	nodePool := v1.NodePool{
		Name: createdNodePool.Name,
	}

	return &nodePool, nil
}
