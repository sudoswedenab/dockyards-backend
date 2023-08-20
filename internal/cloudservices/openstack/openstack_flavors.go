package openstack

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
)

func (s *openStackService) GetFlavorNodePool(flavorID string) (*model.NodePool, error) {
	computev2, err := openstack.NewComputeV2(s.providerClient, gophercloud.EndpointOpts{Region: s.region})
	if err != nil {
		return nil, err
	}

	flavor, err := flavors.Get(computev2, flavorID).Extract()
	if err != nil {
		return nil, err
	}

	nodePool := model.NodePool{
		CPUCount:   flavor.VCPUs,
		RAMSizeMB:  flavor.RAM,
		DiskSizeGB: flavor.Disk,
	}

	return &nodePool, nil
}
