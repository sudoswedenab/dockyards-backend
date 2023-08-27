package openstack

import (
	"errors"
	"math"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/networks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
)

func (s *openStackService) PrepareEnvironment(organization *v1.Organization, cluster *v1.Cluster, nodePoolOptions *v1.NodePoolOptions) (*types.CloudConfig, error) {
	logger := s.logger.With("node-pool", nodePoolOptions.Name, "cluster", cluster.Name, "organization", organization.Name)

	openStackOrganization, err := s.getOpenStackOrganization(organization)
	if err != nil {
		logger.Error("error getting openstack organization", "err", err)
		return nil, err
	}

	logger.Debug("got openstack organization", "id", openStackOrganization.ID, "project", openStackOrganization.OpenStackProject.OpenStackID)

	scopedClient, err := s.getScopedClient(openStackOrganization.OpenStackProject.OpenStackID)
	if err != nil {
		return nil, err
	}

	computev2, err := openstack.NewComputeV2(scopedClient, gophercloud.EndpointOpts{Region: s.region})
	if err != nil {
		return nil, err
	}

	networkv2, err := openstack.NewNetworkV2(scopedClient, gophercloud.EndpointOpts{Region: s.region})
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

	networkLabel := "default"
	if nodePoolOptions.LoadBalancer != nil && *nodePoolOptions.LoadBalancer {
		networkLabel = "elasticip"
	}

	var netID string
	for _, network := range allNetworks {
		logger.Debug("checking network", "id", network.ID)
		if network.Label == networkLabel {
			logger.Debug("found network to use", "id", network.ID, "label", network.Label)
			netID = network.ID
			break
		}
	}
	if netID == "" {
		return nil, errors.New("unable to find suitable network")
	}

	keypairName := names.EncodeName(organization.Name, names.EncodeName(cluster.Name, nodePoolOptions.Name))
	createOpts := keypairs.CreateOpts{
		Name: keypairName,
	}

	keypair, err := keypairs.Create(computev2, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	s.logger.Debug("created keypair", "name", keypair.Name)

	securityGroups := []string{}

	secgroupName := cluster.Name + "-" + nodePoolOptions.Name
	secgroupOpts := secgroups.CreateOpts{
		Name:        secgroupName,
		Description: "no",
	}

	securityGroup, err := secgroups.Create(computev2, secgroupOpts).Extract()
	if err != nil {
		s.logger.Error("error preparing environment", "err", err)

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
			s.logger.Error("error preparing environment", "err", err)

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

	logger.Debug("openstack cloud config created")

	return &config, nil
}

func (s *openStackService) CleanEnvironment(organization *v1.Organization, config *types.CloudConfig) error {
	openStackOrganization, err := s.getOpenStackOrganization(organization)
	if err != nil {
		s.logger.Error("error getting openstack organization", "err", err)

		return err
	}

	s.logger.Debug("cleaning environment", "id", openStackOrganization.ID, "project", openStackOrganization.OpenStackProject.OpenStackID)

	scopedClient, err := s.getScopedClient(openStackOrganization.OpenStackProject.OpenStackID)

	computev2, err := openstack.NewComputeV2(scopedClient, gophercloud.EndpointOpts{Region: s.region})
	if err != nil {
		s.logger.Error("error creating compute service client", "err", err)

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
			TenantID: openStackOrganization.OpenStackProject.OpenStackID,
		}

		s.addGarbage(&securityGroup)
	}

	return nil
}

func (s *openStackService) getClosestFlavorID(flavors []flavors.Flavor, nodePoolOptions *v1.NodePoolOptions) string {
	closestFlavorID := ""
	shortestDistance := math.MaxFloat64

	if nodePoolOptions.CPUCount == nil {
		nodePoolOptions.CPUCount = util.Ptr(0)
	}

	if nodePoolOptions.RAMSizeMb == nil {
		nodePoolOptions.RAMSizeMb = util.Ptr(0)
	}

	if nodePoolOptions.DiskSizeGb == nil {
		nodePoolOptions.DiskSizeGb = util.Ptr(0)
	}

	s.logger.Debug("flavor requirements", "ram", *nodePoolOptions.RAMSizeMb, "cpu", *nodePoolOptions.CPUCount, "disk", *nodePoolOptions.DiskSizeGb)

	for _, flavor := range flavors {
		diskSquared := math.Pow(float64(flavor.Disk-*nodePoolOptions.DiskSizeGb), 2)
		ramSquared := math.Pow(float64(flavor.RAM-*nodePoolOptions.RAMSizeMb), 2)
		vcpuSquared := math.Pow(float64(flavor.VCPUs-*nodePoolOptions.CPUCount), 2)

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
