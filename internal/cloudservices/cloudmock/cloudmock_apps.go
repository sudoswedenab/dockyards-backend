package cloudmock

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
)

func (s *MockCloudService) GetClusterApps(organization *model.Organization, cluster *model.Cluster) (*[]model.App, error) {
	apps := []model.App{}
	for key := range s.apps {
		apps = append(apps, *s.apps[key])
	}

	return &apps, nil
}
