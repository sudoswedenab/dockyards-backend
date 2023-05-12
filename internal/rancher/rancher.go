package rancher

import (
	"strings"
	"sync"

	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/rancher/norman/clientbase"
	normanTypes "github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"golang.org/x/exp/slog"
)

type rancher struct {
	managementClient *managementv3.Client
	url              string
	bearerToken      string
	logger           *slog.Logger
	providerClient   *gophercloud.ProviderClient
	authInfo         *clientconfig.AuthInfo
	garbageMutex     *sync.Mutex
	garbageObjects   map[string]*normanTypes.Resource
}

var _ types.ClusterService = &rancher{}

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

	r := rancher{
		managementClient: managementClient,
		bearerToken:      bearerToken,
		url:              url,
		logger:           logger,
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

func encodeString(s string) string {
	return strings.ReplaceAll(s, "-", "--")
}

func decodeString(s string) string {
	return strings.ReplaceAll(s, "--", "-")
}

func encodeName(org, cluster string) string {
	encodedOrg := encodeString(org)
	encodedCluster := encodeString(cluster)
	return encodedOrg + "-" + encodedCluster
}

func decodeName(s string) (string, string) {
	var split [2]string
	i := 0
	t := len(s) - 1
	for i < t {
		if s[i] == '-' {
			if s[i-1] != '-' && s[i+1] != '-' {
				split[0] = s[0:i]
				split[1] = s[i+1:]
				break
			}
		}
		i += 1
	}

	// name has no dash in it, name is for a cluster without org
	if split[0] == "" {
		return "", s
	}

	decodedOrg := decodeString(split[0])
	decodedName := decodeString(split[1])

	return decodedOrg, decodedName
}
