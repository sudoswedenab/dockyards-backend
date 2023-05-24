package openstack

import (
	"errors"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"gorm.io/gorm"
)

func (s *openStackService) createApplicationCredential(organization *model.Organization, projectID string) (*applicationcredentials.ApplicationCredential, error) {
	scope := gophercloud.AuthScope{
		ProjectID: projectID,
	}

	authOptions := gophercloud.AuthOptions{
		IdentityEndpoint:            s.authOptions.IdentityEndpoint,
		Username:                    s.authOptions.Username,
		UserID:                      s.authOptions.UserID,
		Password:                    s.authOptions.Password,
		Passcode:                    s.authOptions.Passcode,
		DomainID:                    s.authOptions.DomainID,
		DomainName:                  s.authOptions.DomainName,
		AllowReauth:                 s.authOptions.AllowReauth,
		TokenID:                     s.authOptions.TokenID,
		ApplicationCredentialID:     s.authOptions.ApplicationCredentialID,
		ApplicationCredentialName:   s.authOptions.ApplicationCredentialName,
		ApplicationCredentialSecret: s.authOptions.ApplicationCredentialSecret,
		TenantID:                    s.authOptions.TenantID,
		TenantName:                  s.authOptions.TenantName,
		Scope:                       &scope,
	}

	providerClient, err := openstack.AuthenticatedClient(authOptions)
	if err != nil {
		return nil, err
	}

	s.logger.Debug("created new provider client", "scope", scope)

	authResult := providerClient.GetAuthResult()
	createResult := authResult.(tokens.CreateResult)
	user, err := createResult.ExtractUser()
	if err != nil {
		return nil, err
	}

	identityv3, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{})
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

func (s *openStackService) GetOrganization(organization *model.Organization) (string, error) {
	var openStackProject OpenStackProject
	err := s.db.Joins("LEFT JOIN openstack_organizations ON openstack_projects.id = openstack_organizations.openstack_project_id").Take(&openStackProject, "openstack_organizations.organization_id = ?", organization.ID).Error
	if err != nil {
		return "", err
	}

	s.logger.Debug("got openstack project", "id", openStackProject.OpenStackID)

	return openStackProject.OpenStackID, nil
}
