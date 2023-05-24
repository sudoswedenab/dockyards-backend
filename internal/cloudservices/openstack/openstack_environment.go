package openstack

import (
	"errors"
	"math"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/networks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

func (s *openStackService) PrepareEnvironment(organization *model.Organization, cluster *model.Cluster, nodePoolOptions *model.NodePoolOptions) (*types.CloudConfig, error) {
	logger := s.logger.With("node-pool", nodePoolOptions.Name, "cluster", cluster.Name, "organization", organization.Name)

	openStackOrganization, err := s.getOpenStackOrganization(organization)
	if err != nil {
		logger.Error("error getting openstack organization", "err", err)
		return nil, err
	}

	logger.Debug("got openstack organization", "id", openStackOrganization.ID, "project", openStackOrganization.OpenStackProject.OpenStackID)

	projectAuthInfo := clientconfig.AuthInfo{
		AuthURL:                     s.authOptions.IdentityEndpoint,
		ApplicationCredentialID:     openStackOrganization.ApplicationCredentialID,
		ApplicationCredentialSecret: openStackOrganization.ApplicationCredentialSecret,
	}

	clientOpts := clientconfig.ClientOpts{
		AuthType: clientconfig.AuthV3ApplicationCredential,
		AuthInfo: &projectAuthInfo,
	}

	providerClient, err := clientconfig.AuthenticatedClient(&clientOpts)
	if err != nil {
		s.logger.Error("error creating openstack provider client", "err", err)
		return nil, err
	}

	computev2, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{Region: s.region})
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

	if nodePoolOptions.CPUCount == 0 && nodePoolOptions.RAMSize == 0 && nodePoolOptions.DiskSize == 0 {
		nodePoolOptions.CPUCount = 2
		nodePoolOptions.RAMSize = 4096
		nodePoolOptions.DiskSize = 100
	}

	logger.Debug("flavor requirements", "ram", nodePoolOptions.RAMSize, "cpu", nodePoolOptions.CPUCount, "disk", nodePoolOptions.DiskSize)

	flavorID := s.getClosestFlavorID(allFlavors, nodePoolOptions)
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
		logger.Debug("checking image", "image", image)
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

	var netID string
	for _, network := range allNetworks {
		logger.Debug("checking network", "network", network)
		if network.Label == "default" {
			logger.Debug("found network to use", "id", network.ID, "label", network.Label)
			netID = network.ID
			break
		}
	}
	if netID == "" {
		return nil, errors.New("unable to find suitable network")
	}

	keypairName := cluster.Name + "-" + nodePoolOptions.Name
	createOpts := keypairs.CreateOpts{
		Name: keypairName,
	}

	keypair, err := keypairs.Create(computev2, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	allSecurityGroupPages, err := secgroups.List(computev2).AllPages()
	if err != nil {
		return nil, err
	}

	allSecurityGroups, err := secgroups.ExtractSecurityGroups(allSecurityGroupPages)
	if err != nil {
	}

	securityGroups := []string{}
	for _, securityGroup := range allSecurityGroups {
		logger.Debug("checking security group", "name", securityGroup.Name, "id", securityGroup.ID)
		if securityGroup.Name == "default" {
			securityGroups = append(securityGroups, securityGroup.ID)
		}
	}

	if len(securityGroups) == 0 {
		logger.Debug("could not find a default security group")
	}

	secgroupName := cluster.Name + "-" + nodePoolOptions.Name
	secgroupOpts := secgroups.CreateOpts{
		Name:        secgroupName,
		Description: "no",
	}

	securityGroup, err := secgroups.Create(computev2, secgroupOpts).Extract()
	if err != nil {
		return nil, err
	}

	logger.Debug("created security group", "name", securityGroup.Name, "id", securityGroup.ID)

	securityGroups = append(securityGroups, securityGroup.ID)

	createRuleOpts := secgroups.CreateRuleOpts{
		ParentGroupID: securityGroup.ID,
		FromPort:      1,
		ToPort:        65535,
		IPProtocol:    "TCP",
		CIDR:          "0.0.0.0/0",
	}

	rule, err := secgroups.CreateRule(computev2, createRuleOpts).Extract()
	if err != nil {
		return nil, err
	}

	logger.Debug("created security group rule", "id", rule.ID)

	config := types.CloudConfig{
		AuthURL:                     s.authOptions.IdentityEndpoint,
		ApplicationCredentialID:     openStackOrganization.ApplicationCredentialID,
		ApplicationCredentialSecret: openStackOrganization.ApplicationCredentialSecret,
		FlavorID:                    flavorID,
		ImageID:                     imageID,
		KeypairName:                 keypair.Name,
		NetID:                       netID,
		PrivateKeyFile:              keypair.PrivateKey,
		SecurityGroups:              securityGroups,
	}

	logger.Debug("openstack cloud config created", "config", config)

	return &config, nil
}

func (s *openStackService) CleanEnvironment(config *types.CloudConfig) error {
	computev2, err := openstack.NewComputeV2(s.providerClient, gophercloud.EndpointOpts{Region: s.region})
	if err != nil {
		s.logger.Debug("unexpected error creating service client", "err", err)
		return err
	}

	s.logger.Debug("remove keypair", "name", config.KeypairName)

	err = keypairs.Delete(computev2, config.KeypairName, keypairs.DeleteOpts{}).ExtractErr()
	if err != nil {
		s.logger.Debug("error deleting keypair", "name", config.KeypairName, "err", err)
		return err
	}

	return nil
}

func (s *openStackService) getClosestFlavorID(flavors []flavors.Flavor, nodePoolOptions *model.NodePoolOptions) string {
	closestFlavorID := ""
	shortestDistance := math.MaxFloat64

	for _, flavor := range flavors {
		diskSquared := math.Pow(float64(flavor.Disk-nodePoolOptions.DiskSize), 2)
		ramSquared := math.Pow(float64(flavor.RAM-nodePoolOptions.RAMSize), 2)
		vcpuSquared := math.Pow(float64(flavor.VCPUs-nodePoolOptions.CPUCount), 2)

		distance := math.Sqrt(diskSquared + ramSquared + vcpuSquared)

		s.logger.Debug("checking flavor distance", "id", flavor.ID, "disk", flavor.Disk, "ram", flavor.RAM, "vcpus", flavor.VCPUs, "distance", distance)

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
