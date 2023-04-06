package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/networks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
)

func (r *Rancher) prepareOpenstackEnvironment(cluster *model.Cluster, nodePoolOptions *model.NodePoolOptions) (*openstackConfig, error) {
	logger := r.Logger.With("node-pool", nodePoolOptions.Name, "cluster", cluster.Name)

	computev2, err := openstack.NewComputeV2(r.providerClient, gophercloud.EndpointOpts{Region: "sto1"})
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

	ramSize := nodePoolOptions.RAMSize
	if ramSize == 0 {
		ramSize = 4096
	}

	cpuCount := nodePoolOptions.CPUCount
	if cpuCount == 0 {
		cpuCount = 2
	}

	diskSize := nodePoolOptions.DiskSize
	if diskSize == 0 {
		diskSize = 100
	}

	logger.Debug("flavor requirements", "ram", ramSize, "cpu", cpuCount, "disk", diskSize)

	var flavorID string
	for _, flavor := range allFlavors {
		logger.Debug("checking flavor", "flavor", flavor)
		if flavor.RAM == ramSize && flavor.VCPUs == cpuCount && flavor.Disk == diskSize {
			logger.Debug("found flavor to use", "id", flavor.ID, "name", flavor.Name)
			flavorID = flavor.ID
			break
		}
	}
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

	config := openstackConfig{
		AuthURL:                     r.authInfo.AuthURL,
		ApplicationCredentialID:     r.authInfo.ApplicationCredentialID,
		ApplicationCredentialSecret: r.authInfo.ApplicationCredentialSecret,
		FlavorID:                    flavorID,
		ImageID:                     imageID,
		KeypairName:                 keypair.Name,
		NetID:                       netID,
		PrivateKeyFile:              keypair.PrivateKey,
		SecGroups:                   "default,arst",
		SSHUser:                     "ubuntu",
	}

	logger.Debug("openstack config created", "config", config)

	return &config, nil
}
