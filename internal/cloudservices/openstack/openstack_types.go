package openstack

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type openStackService struct {
	authOptions    *gophercloud.AuthOptions
	providerClient *gophercloud.ProviderClient
	logger         *slog.Logger
	region         string
	db             *gorm.DB
}

var _ types.CloudService = &openStackService{}

type OpenStackOption func(*openStackService)

type OpenStackProject struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	OpenStackID string    `gorm:"column:openstack_id"`
}

func (p *OpenStackProject) TableName() string {
	return "openstack_projects"
}

type OpenStackOrganization struct {
	ID                 uuid.UUID `gorm:"primaryKey"`
	OpenStackProjectID uuid.UUID `gorm:"column:openstack_project_id"`
	OpenStackProject   OpenStackProject
	OrganizationID     uuid.UUID
	Organization       model.Organization
}

func (o *OpenStackOrganization) TableName() string {
	return "openstack_organizations"
}
