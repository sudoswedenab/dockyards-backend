package openstack

import (
	"log/slog"
	"sync"

	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"github.com/gophercloud/gophercloud"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type openStackService struct {
	authOptions      *gophercloud.AuthOptions
	providerClient   *gophercloud.ProviderClient
	logger           *slog.Logger
	scopedClients    map[string]*gophercloud.ProviderClient
	garbageObjects   map[string]any
	garbageMutex     *sync.Mutex
	endpointOpts     gophercloud.EndpointOpts
	insecureLogging  bool
	controllerClient client.Client
}

var _ cloudservices.CloudService = &openStackService{}

type OpenStackOption func(*openStackService)
