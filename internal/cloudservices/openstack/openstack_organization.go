package openstack

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"gorm.io/gorm"
)

func (s *openStackService) createApplicationCredential(organization *v1.Organization, projectID string) (*applicationcredentials.ApplicationCredential, error) {
	scopedClient, err := s.getScopedClient(projectID)
	if err != nil {
		return nil, err
	}

	authResult := scopedClient.GetAuthResult()
	createResult := authResult.(tokens.CreateResult)
	user, err := createResult.ExtractUser()
	if err != nil {
		return nil, err
	}

	identityv3, err := openstack.NewIdentityV3(scopedClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	s.logger.Debug("created new identity service client", "endpoint", identityv3.Endpoint)

	createOpts := applicationcredentials.CreateOpts{
		Name: organization.Name,
	}

	applicationCredential, err := applicationcredentials.Create(identityv3, user.ID, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	s.logger.Debug("created new application credential", "id", applicationCredential.ID)

	return applicationCredential, nil

}

func (s *openStackService) CreateOrganization(organization *v1.Organization) (string, error) {
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

	applicationCredential, err := s.createApplicationCredential(organization, openStackProject.OpenStackID)
	if err != nil {
		s.logger.Error("error creating application credential", "err", err)
		return "", err
	}

	openStackOrganization := OpenStackOrganization{
		ID:                          uuid.New(),
		OpenStackProjectID:          openStackProject.ID,
		OrganizationID:              organization.ID,
		ApplicationCredentialID:     applicationCredential.ID,
		ApplicationCredentialSecret: applicationCredential.Secret,
	}

	err = s.db.Create(&openStackOrganization).Error
	if err != nil {
		s.logger.Error("error creating openstack organization in database", "err", err)
		return "", err
	}

	return openStackProject.OpenStackID, nil
}

func (s *openStackService) GetOrganization(organization *v1.Organization) (string, error) {
	var openStackProject OpenStackProject
	err := s.db.Joins("LEFT JOIN openstack_organizations ON openstack_projects.id = openstack_organizations.openstack_project_id").Take(&openStackProject, "openstack_organizations.organization_id = ?", organization.ID).Error
	if err != nil {
		return "", err
	}

	s.logger.Debug("got openstack project", "id", openStackProject.OpenStackID)

	return openStackProject.OpenStackID, nil
}

func (s *openStackService) getOpenStackOrganization(organization *v1.Organization) (*OpenStackOrganization, error) {
	var openStackOrganization OpenStackOrganization
	err := s.db.Preload("OpenStackProject").Take(&openStackOrganization, "organization_id = ?", organization.ID).Error
	if err != nil {
		return nil, err
	}

	return &openStackOrganization, nil
}

func (s *openStackService) DeleteOrganization(organization *v1.Organization) error {
	openStackOrganization, err := s.getOpenStackOrganization(organization)
	if err != nil {
		return err
	}

	scopedClient, err := s.getScopedClient(openStackOrganization.OpenStackProject.OpenStackID)
	if err != nil {
		return err
	}

	authResult := scopedClient.GetAuthResult()
	createResult := authResult.(tokens.CreateResult)
	user, err := createResult.ExtractUser()
	if err != nil {
		return err
	}

	s.logger.Debug("deleting application credential", "user", user.ID, "id", openStackOrganization.ApplicationCredentialID)

	identityv3, err := openstack.NewIdentityV3(scopedClient, s.endpointOpts)
	if err != nil {
		return err
	}

	err = applicationcredentials.Delete(identityv3, user.ID, openStackOrganization.ApplicationCredentialID).ExtractErr()
	if err != nil {
		return err
	}

	s.logger.Debug("deleted application credential", "id", openStackOrganization.ApplicationCredentialID)

	s.logger.Debug("deleting openstack organization binding", "id", openStackOrganization.ID)

	err = s.db.Delete(&openStackOrganization, "id = ?", openStackOrganization.ID).Error
	if err != nil {
		return err
	}

	s.logger.Debug("deleted openstack organization binding", "id", openStackOrganization.ID)

	return nil
}
