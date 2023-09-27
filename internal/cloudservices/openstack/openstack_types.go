package openstack

import (
	"log/slog"
	"sync"

	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/ipam"
	"github.com/gophercloud/gophercloud"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type openStackService struct {
	authOptions      *gophercloud.AuthOptions
	providerClient   *gophercloud.ProviderClient
	logger           *slog.Logger
	db               *gorm.DB
	scopedClients    map[string]*gophercloud.ProviderClient
	garbageObjects   map[string]any
	garbageMutex     *sync.Mutex
	endpointOpts     gophercloud.EndpointOpts
	insecureLogging  bool
	ipManager        ipam.IPManager
	controllerClient client.Client
}

var _ cloudservices.CloudService = &openStackService{}

type OpenStackOption func(*openStackService)
