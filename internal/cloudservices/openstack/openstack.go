package openstack

import (
	"log/slog"
	"os"
	"sync"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/ipam"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"gorm.io/gorm"
)

func WithLogger(logger *slog.Logger) OpenStackOption {
	return func(s *openStackService) {
		s.logger = logger
	}
}

func WithRegion(region string) OpenStackOption {
	endpointOpts := gophercloud.EndpointOpts{
		Region: region,
	}

	return func(s *openStackService) {
		s.endpointOpts = endpointOpts
	}
}

func WithDatabase(db *gorm.DB) OpenStackOption {
	return func(s *openStackService) {
		s.db = db
	}
}

func WithCloudsYAML(cloud string) OpenStackOption {
	clientOpts := clientconfig.ClientOpts{
		Cloud: cloud,
	}

	authOptions, err := clientconfig.AuthOptions(&clientOpts)
	if err != nil {
		panic(err)
	}

	authOptions.AllowReauth = true

	return func(s *openStackService) {
		s.authOptions = authOptions
	}
}

func WithInsecureLogging(insecureLogging bool) OpenStackOption {
	return func(s *openStackService) {
		s.insecureLogging = insecureLogging
	}
}

func NewOpenStackService(openStackOptions ...OpenStackOption) (*openStackService, error) {
	s := openStackService{
		garbageObjects: make(map[string]any),
		scopedClients:  make(map[string]*gophercloud.ProviderClient),
		garbageMutex:   &sync.Mutex{},
	}

	for _, openStackOption := range openStackOptions {
		openStackOption(&s)
	}

	if s.logger == nil {
		s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
		s.logger.Info("no logger was provided, using default")
	}

	if s.endpointOpts.Region == "" {
		s.logger.Debug("using default region", "region", "sto1")

		s.endpointOpts.Region = "sto1"
	}

	providerClient, err := openstack.AuthenticatedClient(*s.authOptions)
	if err != nil {
		return nil, err
	}

	s.providerClient = providerClient

	ipManagerOptions := []ipam.ManagerOption{
		ipam.WithLogger(s.logger),
		ipam.WithDB(s.db),
	}

	ipManager, err := ipam.NewIPManager(ipManagerOptions...)
	if err != nil {
		return nil, err
	}
	s.ipManager = ipManager

	if s.insecureLogging {
		s.logger.Warn("insecure logging allowed")
	}

	return &s, nil
}
