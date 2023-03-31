package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/rancher/norman/clientbase"
	"github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"golang.org/x/exp/slog"
)

type RancherService interface {
	RancherCreateUser(model.RancherUser) (string, error)
	RancherCreateCluster(model.ClusterOptions) (managementv3.Cluster, error)
	RancherCreateNodePool(model.ClusterOptions, string) (managementv3.NodePool, error)
	RancherLogin(model.User) (string, error)
	GetClusters(opts *types.ListOpts) (managementv3.ClusterCollection, error)
	DeleteCluster(string) error
	CreateClusterRole() error
	GetSupportedVersions() []string
}

type Rancher struct {
	ManagementClient *managementv3.Client
	url              string
	bearerToken      string
	Logger           *slog.Logger
}

var _ RancherService = &Rancher{}

func NewRancher(bearerToken, url string, logger *slog.Logger, trustInsecure bool) (RancherService, error) {
	clientOpts := clientbase.ClientOpts{
		URL:      url,
		TokenKey: bearerToken,
		Insecure: trustInsecure,
	}

	managementClient, err := managementv3.NewClient(&clientOpts)
	if err != nil {
		return nil, err
	}

	r := Rancher{
		ManagementClient: managementClient,
		bearerToken:      bearerToken,
		url:              url,
		Logger:           logger,
	}

	return &r, err
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(b int64) *int64 {
	return &b
}
