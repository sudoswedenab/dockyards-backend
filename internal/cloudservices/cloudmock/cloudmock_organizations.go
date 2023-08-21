package cloudmock

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
)

func (s *MockCloudService) DeleteOrganization(organization *model.Organization) error {
	_, hasOrganization := s.organizations[organization.Name]
	if !hasOrganization {
		return errors.New("no such organization")
	}

	return nil
}
