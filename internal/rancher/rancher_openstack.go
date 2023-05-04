package rancher

import (
	"errors"
	"math"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/networks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
)

func (r *Rancher) prepareOpenstackEnvironment(cluster *model.Cluster, nodePoolOptions *model.NodePoolOptions) (*openstackConfig, error) {
	logger := r.Logger.With("node-pool", nodePoolOptions.Name, "cluster", cluster.Name, "organization", cluster.Organization)

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

	encodedName := encodeName(cluster.Organization, cluster.Name)

	keypairName := encodedName + "-" + nodePoolOptions.Name
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

func (r *Rancher) cleanOpenstackEnvironment(config *openstackConfig) error {
	computev2, err := openstack.NewComputeV2(r.providerClient, gophercloud.EndpointOpts{Region: "sto1"})
	if err != nil {
		r.Logger.Debug("unexpected error creating service client", "err", err)
		return err
	}

	r.Logger.Debug("remove keypair", "name", config.KeypairName)

	err = keypairs.Delete(computev2, config.KeypairName, keypairs.DeleteOpts{}).ExtractErr()
	if err != nil {
		r.Logger.Debug("error deleting keypair", "name", config.KeypairName, "err", err)
		return err
	}

	return nil
}

func (r *Rancher) getClosestFlavorID(flavors []flavors.Flavor, nodePoolOptions *model.NodePoolOptions) string {
	closestFlavorID := ""
	shortestDistance := math.MaxFloat64

	for _, flavor := range flavors {
		diskSquared := math.Pow(float64(flavor.Disk-nodePoolOptions.DiskSize), 2)
		ramSquared := math.Pow(float64(flavor.RAM-nodePoolOptions.RAMSize), 2)
		vcpuSquared := math.Pow(float64(flavor.VCPUs-nodePoolOptions.CPUCount), 2)

		distance := math.Sqrt(diskSquared + ramSquared + vcpuSquared)

		r.Logger.Debug("checking flavor distance", "id", flavor.ID, "disk", flavor.Disk, "ram", flavor.RAM, "vcpus", flavor.VCPUs, "distance", distance)

		if distance == 0 {
			closestFlavorID = flavor.ID
			break
		}

		if distance < shortestDistance {
			shortestDistance = distance
			closestFlavorID = flavor.ID
		}
	}

	r.Logger.Debug("found flavor to use", "id", closestFlavorID)

	return closestFlavorID
}
