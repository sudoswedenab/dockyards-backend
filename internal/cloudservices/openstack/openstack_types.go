package openstack

import (
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type openStackService struct {
	authInfo       *clientconfig.AuthInfo
	providerClient *gophercloud.ProviderClient
	logger         *slog.Logger
	region         string
	db             *gorm.DB
}

var _ types.CloudService = &openStackService{}

type OpenStackOption func(*openStackService)
