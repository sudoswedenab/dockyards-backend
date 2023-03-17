package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/rancher/norman/clientbase"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type RancherService interface {
	RancherCreateUser(model.RancherUser) (string, error)
	RancherCreateCluster(string, string, string, string) (managementv3.Cluster, error)
	RancherCreateNodePool(string, string) (managementv3.NodePool, error)
	RancherLogin(model.User) (string, error)
	CreateClusterRole() error
}

type Rancher struct {
	ManagementClient *managementv3.Client
	url              string
	bearerToken      string
}

var _ RancherService = &Rancher{}

func NewRancher(bearerToken, url string) (RancherService, error) {
	clientOpts := clientbase.ClientOpts{
		URL: url,
	}
	managementClient, err := managementv3.NewClient(&clientOpts)
	if err != nil {
		return nil, err
	}

	r := Rancher{
		ManagementClient: managementClient,
		bearerToken:      bearerToken,
		url:              url,
	}

	return &r, err
}
