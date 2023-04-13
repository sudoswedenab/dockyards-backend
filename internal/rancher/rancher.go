package rancher

import (
	"sync"

	"bitbucket.org/sudosweden/backend/internal/types"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/rancher/norman/clientbase"
	normanTypes "github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"golang.org/x/exp/slog"
)

type Rancher struct {
	ManagementClient *managementv3.Client
	url              string
	bearerToken      string
	Logger           *slog.Logger
	providerClient   *gophercloud.ProviderClient
	authInfo         *clientconfig.AuthInfo
	garbageMutex     *sync.Mutex
	garbageObjects   map[string]*normanTypes.Resource
}

var _ types.ClusterService = &Rancher{}

func NewRancher(bearerToken, url string, logger *slog.Logger, trustInsecure bool, authURL, appID, appSecret string) (types.ClusterService, error) {
	clientOpts := clientbase.ClientOpts{
		URL:      url,
		TokenKey: bearerToken,
		Insecure: trustInsecure,
	}

	managementClient, err := managementv3.NewClient(&clientOpts)
	if err != nil {
		return nil, err
	}

	authInfo := &clientconfig.AuthInfo{
		AuthURL:                     authURL,
		ApplicationCredentialID:     appID,
		ApplicationCredentialSecret: appSecret,
		AllowReauth:                 true,
	}

	openstackOpts := clientconfig.ClientOpts{
		AuthType: clientconfig.AuthV3ApplicationCredential,
		AuthInfo: authInfo,
	}

	providerClient, err := clientconfig.AuthenticatedClient(&openstackOpts)
	if err != nil {
		return nil, err
	}

	r := Rancher{
		ManagementClient: managementClient,
		bearerToken:      bearerToken,
		url:              url,
		Logger:           logger,
		providerClient:   providerClient,
		authInfo:         authInfo,
		garbageMutex:     &sync.Mutex{},
		garbageObjects:   make(map[string]*normanTypes.Resource),
	}

	return &r, err
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(b int64) *int64 {
	return &b
}
