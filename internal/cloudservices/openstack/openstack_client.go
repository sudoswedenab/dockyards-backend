package openstack

import (
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

func (s *openStackService) getScopedClient(projectID string) (*gophercloud.ProviderClient, error) {
	if projectID == "" {
		return nil, errors.New("project id must not be empty")

	}

	scopedClient, hasScopedClient := s.scopedClients[projectID]
	if !hasScopedClient {
		s.logger.Debug("creating scoped provider client", "id", projectID)

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

		var err error
		scopedClient, err = openstack.AuthenticatedClient(authOptions)
		if err != nil {
			return nil, err
		}

		s.scopedClients[projectID] = scopedClient
	}

	s.logger.Debug("returning scoped provider client", "id", projectID, "client", fmt.Sprintf("%p", scopedClient))

	return scopedClient, nil
}
