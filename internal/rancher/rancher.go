package rancher

import (
	"strings"
	"sync"

	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/rancher/norman/clientbase"
	normanTypes "github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"golang.org/x/exp/slog"
)

type rancher struct {
	managementClient *managementv3.Client
	clientOpts       *clientbase.ClientOpts
	logger           *slog.Logger
	garbageMutex     *sync.Mutex
	garbageObjects   map[string]*normanTypes.Resource
	cloudService     types.CloudService
}

var _ types.ClusterService = &rancher{}

type RancherOption func(*rancher)

func WithLogger(logger *slog.Logger) RancherOption {
	return func(r *rancher) {
		r.logger = logger
	}
}

func WithRancherClientOpts(url, tokenKey string, insecure bool) RancherOption {
	return func(r *rancher) {
		r.clientOpts = &clientbase.ClientOpts{
			URL:      url,
			TokenKey: tokenKey,
			Insecure: insecure,
		}
	}
}

func WithCloudService(cloudService types.CloudService) RancherOption {
	return func(r *rancher) {
		r.cloudService = cloudService
	}
}

func NewRancher(rancherOptions ...RancherOption) (types.ClusterService, error) {
	r := rancher{
		garbageMutex:   &sync.Mutex{},
		garbageObjects: make(map[string]*normanTypes.Resource),
	}

	for _, rancherOption := range rancherOptions {
		rancherOption(&r)
	}

	managementClient, err := managementv3.NewClient(r.clientOpts)
	if err != nil {
		return nil, err
	}
	r.managementClient = managementClient

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
