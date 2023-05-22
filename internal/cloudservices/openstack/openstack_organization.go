package openstack

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (s *openStackService) CreateOrganization(organization *model.Organization) (string, error) {
	var openStackProject OpenStackProject
	err := s.db.Joins("LEFT OUTER JOIN openstack_organizations ON openstack_projects.id = openstack_organizations.openstack_project_id").Take(&openStackProject, "openstack_organizations.openstack_project_id IS NULL").Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Debug("no openstack projects available for use")
			return "", errors.New("no openstack projects available for use")
		}

		s.logger.Error("error fetching openstack project from database", "err", err)
		return "", err
	}

	s.logger.Debug("took openstack project", "id", openStackProject.ID, "openstack", openStackProject.OpenStackID)

	openStackOrganization := OpenStackOrganization{
		ID:                 uuid.New(),
		OpenStackProjectID: openStackProject.ID,
		OrganizationID:     organization.ID,
	}
	err = s.db.Create(&openStackOrganization).Error
	if err != nil {
		s.logger.Error("error creating openstack organization in database", "err")
		return "", err
	}

	return openStackProject.OpenStackID, nil
}

func (s *openStackService) GetOrganization(organization *model.Organization) (string, error) {
	var openStackProject OpenStackProject
	err := s.db.Joins("LEFT JOIN openstack_organizations ON openstack_projects.id = openstack_organizations.openstack_project_id").Take(&openStackProject, "openstack_organizations.organization_id = ?", organization.ID).Error
	if err != nil {
		return "", err
	}

	s.logger.Debug("got openstack project", "id", openStackProject.OpenStackID)

	return openStackProject.OpenStackID, nil
}
