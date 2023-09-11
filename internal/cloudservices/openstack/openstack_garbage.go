package openstack

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
)

func (s *openStackService) addGarbage(g any) {
	garbageID := uuid.New().String()

	switch v := g.(type) {
	case *secgroups.SecurityGroup:
		s.garbageMutex.Lock()
		s.garbageObjects[garbageID] = v
		s.garbageMutex.Unlock()
	default:
		s.logger.Warn("ignoring unsupported garbage type", "type", fmt.Sprintf("%T", g))
	}
}

func (s *openStackService) DeleteGarbage() {
	s.logger.Debug("delete garbage start")

	s.garbageMutex.Lock()

	for garbageID, g := range s.garbageObjects {
		switch garbageObject := g.(type) {
		case *secgroups.SecurityGroup:
			logger := s.logger.With("id", garbageObject.ID, "tenant", garbageObject.TenantID)

			logger.Debug("deleting security group")

			scopedClient, err := s.getScopedClient(garbageObject.TenantID)
			if err != nil {
				logger.Warn("error getting reusable client", "err", err)

				break
			}

			computev2, err := openstack.NewComputeV2(scopedClient, s.endpointOpts)
			if err != nil {
				logger.Warn("error creating compute service client", "err", err)

				break
			}

			err = secgroups.Delete(computev2, garbageObject.ID).ExtractErr()
			if err != nil {
				logger.Warn("error deleteting security group", "err", err)

				break
			}

			delete(s.garbageObjects, garbageID)

			logger.Debug("deleted security group")
		default:
			s.logger.Warn("unknown garbage object type", "type", fmt.Sprintf("%T", garbageObject))
		}
	}

	s.garbageMutex.Unlock()

	s.logger.Debug("delete garbage end")
}
