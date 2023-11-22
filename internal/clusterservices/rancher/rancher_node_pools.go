package rancher

import (
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	corev1 "k8s.io/api/core/v1"
)

func (r *rancher) CreateNodePool(organization *v1alpha2.Organization, cluster *v1alpha1.Cluster, nodePool *v1alpha1.NodePool) (*v1alpha1.NodePoolStatus, error) {
	cloudConfig, err := r.cloudService.PrepareEnvironment(organization, cluster, nodePool)
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

	if nodePool.Spec.LoadBalancer {
		taint := managementv3.Taint{
			Effect: string(corev1.TaintEffectNoSchedule),
			Key:    TaintNodeRoleLoadBalancer,
		}
		nodeTaints = append(nodeTaints, taint)

		nodeLabels[LabelNodeRoleLoadBalancer] = ""
	}

	nodeTemplateName := nodePool.Name
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

	quantity := int64(1)
	if nodePool.Spec.Replicas != nil {
		quantity = int64(*nodePool.Spec.Replicas)
	}

	hostnamePrefix := nodePool.Name + "-"
	opts := managementv3.NodePool{
		ClusterID:               cluster.Status.ClusterServiceID,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		HostnamePrefix:          hostnamePrefix,
		Name:                    nodePool.Name,
		NodeTaints:              nodeTaints,
		NodeTemplateID:          createdNodeTemplate.ID,
		Quantity:                quantity,
		Worker:                  !nodePool.Spec.DedicatedRole,
		ControlPlane:            nodePool.Spec.ControlPlane,
		Etcd:                    nodePool.Spec.ControlPlane,
	}

	createdNodePool, err := r.managementClient.NodePool.Create(&opts)
	if err != nil {
		return nil, err
	}

	nodePoolStatus := v1alpha1.NodePoolStatus{
		ClusterServiceID: createdNodePool.ID,
	}

	return &nodePoolStatus, nil
}

func (r *rancher) GetNodePool(nodePoolID string) (*v1alpha1.NodePoolStatus, error) {
	rancherNodePool, err := r.managementClient.NodePool.ByID(nodePoolID)
	if err != nil {
		return nil, err
	}

	nodePoolStatus := v1alpha1.NodePoolStatus{
		ClusterServiceID: rancherNodePool.ID,
	}

	return &nodePoolStatus, nil
}

func (r *rancher) DeleteNodePool(organization *v1alpha2.Organization, nodePoolID string) error {
	nodePool, err := r.managementClient.NodePool.ByID(nodePoolID)
	if err != nil {
		return err
	}

	var customNodeTemplate CustomNodeTemplate
	err = r.managementClient.ByID(managementv3.NodeTemplateType, nodePool.NodeTemplateID, &customNodeTemplate)
	if err != nil {
		r.logger.Warn("error fetching node template by id", "id", nodePool.NodeTemplateID)

		return err
	}

	r.logger.Debug("node pool custom node template", "id", customNodeTemplate.ID)

	securityGroups := strings.Split(customNodeTemplate.OpenstackConfig.SecGroups, ",")

	cloudConfig := cloudservices.CloudConfig{
		KeypairName:    customNodeTemplate.OpenstackConfig.KeypairName,
		SecurityGroups: securityGroups,
		NetID:          customNodeTemplate.OpenstackConfig.NetID,
	}

	err = r.cloudService.CleanEnvironment(organization, &cloudConfig)
	if err != nil {
		r.logger.Warn("error cleaning openstack environment", "err", err)
		return err
	}

	err = r.managementClient.NodePool.Delete(nodePool)
	if err != nil {
		r.logger.Warn("error deleting node pool", "err", err)
		return err
	}
	r.logger.Debug("deleted node pool", "id", nodePool.ID, "name", nodePool.Name)

	// node template cannot be deleted at this point
	// add it to the garbage
	r.addGarbage(&customNodeTemplate.NodeTemplate.Resource)

	r.logger.Debug("added node template to garbage", "id", customNodeTemplate.ID)

	return nil
}
