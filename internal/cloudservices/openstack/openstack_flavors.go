package openstack

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
)

func (s *openStackService) GetFlavorNodePool(flavorID string) (*v1.NodePool, error) {
	computev2, err := openstack.NewComputeV2(s.providerClient, s.endpointOpts)
	if err != nil {
		return nil, err
	}

	flavor, err := flavors.Get(computev2, flavorID).Extract()
	if err != nil {
		return nil, err
	}

	nodePool := v1.NodePool{
		CPUCount:   flavor.VCPUs,
		RAMSizeMb:  flavor.RAM,
		DiskSizeGb: flavor.Disk,
	}

	return &nodePool, nil
}
