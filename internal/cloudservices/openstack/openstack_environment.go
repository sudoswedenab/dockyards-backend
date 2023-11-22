package openstack

import (
	"context"
	"errors"
	"log/slog"
	"math"

	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/networks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	corev1 "k8s.io/api/core/v1"
)

func (s *openStackService) PrepareEnvironment(organization *v1alpha2.Organization, cluster *v1alpha1.Cluster, nodePool *v1alpha1.NodePool) (*cloudservices.CloudConfig, error) {
	logger := s.logger.With("nodepool", nodePool.Name, "cluster", cluster.Name, "organization", organization.Name)

	openstackProject, err := s.getOpenstackProject(organization)
	if err != nil {
		return nil, err
	}

	secret, err := s.getOpenstackSecret(organization)
	if err != nil {
		return nil, err
	}

	logger.Debug("got openstack project", "uid", openstackProject.UID, "project", openstackProject.Spec.ProjectID)

	scopedClient, err := s.getScopedClient(openstackProject.Spec.ProjectID)
	if err != nil {
		return nil, err
	}

	computev2, err := openstack.NewComputeV2(scopedClient, s.endpointOpts)
	if err != nil {
		return nil, err
	}

	networkv2, err := openstack.NewNetworkV2(scopedClient, s.endpointOpts)
	if err != nil {
		return nil, err
	}

	flavorListOpts := flavors.ListOpts{
		MinRAM:  4,
		MinDisk: 100,
	}

	allFlavorPages, err := flavors.ListDetail(computev2, flavorListOpts).AllPages()
	if err != nil {
		return nil, err
	}

	allFlavors, err := flavors.ExtractFlavors(allFlavorPages)
	if err != nil {
		return nil, err
	}

	flavorID := s.getClosestFlavorID(allFlavors, nodePool)
	if flavorID == "" {
		return nil, errors.New("unable to find a suitable flavor")
	}

	imageListOpts := images.ListOpts{
		Name: "ubuntu-22.04",
	}

	allImagePages, err := images.ListDetail(computev2, imageListOpts).AllPages()
	if err != nil {
		return nil, err
	}

	allImages, err := images.ExtractImages(allImagePages)
	if err != nil {
		return nil, err
	}

	var imageID string
	for _, image := range allImages {
		s.logger.Log(context.Background(), slog.LevelDebug-1, "checking image", "image", image)

		if image.Name == "ubuntu-22.04" {
			logger.Debug("found image to use", "id", image.ID, "name", image.Name)

			imageID = image.ID
			break
		}
	}
	if imageID == "" {
		return nil, errors.New("unable to find suitable image")
	}

	allNetworkPages, err := networks.List(computev2).AllPages()
	if err != nil {
		return nil, err
	}

	allNetworks, err := networks.ExtractNetworks(allNetworkPages)
	if err != nil {
		return nil, err
	}

	networkLabel := "default"
	if nodePool.Spec.LoadBalancer {
		networkLabel = "elasticip"
	}

	var netID string
	for _, network := range allNetworks {
		s.logger.Log(context.Background(), slog.LevelDebug-1, "checking network", "id", network.ID)

		if network.Label == networkLabel {
			logger.Debug("found network to use", "id", network.ID, "label", network.Label)

			netID = network.ID
			break
		}
	}
	if netID == "" {
		return nil, errors.New("unable to find suitable network")
	}

	keypairName := name.EncodeName(organization.Name, name.EncodeName(cluster.Name, nodePool.Name))
	createOpts := keypairs.CreateOpts{
		Name: keypairName,
	}

	keypair, err := keypairs.Create(computev2, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	s.logger.Debug("created keypair", "name", keypair.Name)

	securityGroups := []string{}

	secgroupOpts := secgroups.CreateOpts{
		Name:        nodePool.Name,
		Description: "no",
	}

	securityGroup, err := secgroups.Create(computev2, secgroupOpts).Extract()
	if err != nil {
		s.logger.Debug("deleting new keypair", "name", keypair.Name)

		deleteErr := keypairs.Delete(computev2, keypair.Name, nil).ExtractErr()
		if deleteErr != nil {
			s.logger.Warn("error deleting new keypair", "err", deleteErr)
		}

		return nil, err
	}

	logger.Debug("created security group", "name", securityGroup.Name, "id", securityGroup.ID)

	securityGroups = append(securityGroups, securityGroup.ID)

	for _, etherType := range []rules.RuleEtherType{rules.EtherType4, rules.EtherType6} {
		createRuleOpts := rules.CreateOpts{
			Direction:  rules.DirIngress,
			EtherType:  etherType,
			SecGroupID: securityGroup.ID,
		}

		rule, err := rules.Create(networkv2, createRuleOpts).Extract()
		if err != nil {
			s.logger.Debug("deleting new security group", "id", securityGroup.ID)

			deleteErr := secgroups.Delete(computev2, securityGroup.ID).ExtractErr()
			if deleteErr != nil {
				s.logger.Warn("error deleting new security group", "err", err)
			}

			s.logger.Debug("deleting new keypair", "name", keypair.Name)

			deleteErr = keypairs.Delete(computev2, keypair.Name, nil).ExtractErr()
			if deleteErr != nil {
				s.logger.Warn("error deleting new keypair", "err", err)
			}

			return nil, err
		}

		logger.Debug("created security group rule", "id", rule.ID)
	}

	applicationCredentialID := secret.Data["applicationCredentialID"]
	applicationCredentialSecret := secret.Data["applicationCredentialSecret"]

	config := cloudservices.CloudConfig{
		AuthURL:                     s.authOptions.IdentityEndpoint,
		ApplicationCredentialID:     string(applicationCredentialID),
		ApplicationCredentialSecret: string(applicationCredentialSecret),
		FlavorID:                    flavorID,
		ImageID:                     imageID,
		KeypairName:                 keypair.Name,
		NetID:                       netID,
		PrivateKeyFile:              keypair.PrivateKey,
		SecurityGroups:              securityGroups,
	}

	logger.Debug("openstack cloud config created")

	return &config, nil
}

func (s *openStackService) CleanEnvironment(organization *v1alpha2.Organization, config *cloudservices.CloudConfig) error {
	openstackProject, err := s.getOpenstackProject(organization)
	if err != nil {
		return err
	}

	s.logger.Debug("cleaning environment", "project", openstackProject.Spec.ProjectID)

	scopedClient, err := s.getScopedClient(openstackProject.Spec.ProjectID)

	computev2, err := openstack.NewComputeV2(scopedClient, s.endpointOpts)
	if err != nil {
		return err
	}

	s.logger.Debug("remove keypair", "name", config.KeypairName)

	err = keypairs.Delete(computev2, config.KeypairName, keypairs.DeleteOpts{}).ExtractErr()
	if err != nil {
		s.logger.Warn("error deleting keypair", "name", config.KeypairName, "err", err)
	}

	for _, securityGroupID := range config.SecurityGroups {
		s.logger.Debug("adding security group to garbage", "id", securityGroupID)

		securityGroup := secgroups.SecurityGroup{
			ID:       securityGroupID,
			TenantID: openstackProject.Spec.ProjectID,
		}

		s.addGarbage(&securityGroup)
	}

	return nil
}

func (s *openStackService) getClosestFlavorID(flavors []flavors.Flavor, nodePool *v1alpha1.NodePool) string {
	closestFlavorID := ""
	shortestDistance := math.MaxFloat64

	vcpus := 0
	resourceCPU, hasResourceCPU := nodePool.Spec.Resources[corev1.ResourceCPU]
	if hasResourceCPU {
		vcpus = int(resourceCPU.Value())
	}

	ram := 0
	resourceMemory, hasResourceMemory := nodePool.Spec.Resources[corev1.ResourceMemory]
	if hasResourceMemory {
		ram = int(resourceMemory.Value() / 1024 / 1024)
	}

	disk := 0
	resourceStorage, hasResourceStorage := nodePool.Spec.Resources[corev1.ResourceStorage]
	if hasResourceStorage {
		disk = int(resourceStorage.Value() / 1024 / 1024 / 1024)
	}

	s.logger.Debug("flavor requirements", "ram", ram, "vcpus", vcpus, "disk", disk)

	for _, flavor := range flavors {
		diskSquared := math.Pow(float64(flavor.Disk-disk), 2)
		ramSquared := math.Pow(float64(flavor.RAM-ram), 2)
		vcpuSquared := math.Pow(float64(flavor.VCPUs-vcpus), 2)

		distance := math.Sqrt(diskSquared + ramSquared + vcpuSquared)

		s.logger.Log(context.Background(), slog.LevelDebug-1, "checking flavor distance", "id", flavor.ID, "disk", flavor.Disk, "ram", flavor.RAM, "vcpus", flavor.VCPUs, "distance", distance)

		if distance == 0 {
			closestFlavorID = flavor.ID
			break
		}

		if distance < shortestDistance {
			shortestDistance = distance
			closestFlavorID = flavor.ID
		}
	}

	s.logger.Debug("found flavor to use", "id", closestFlavorID)

	return closestFlavorID
}
