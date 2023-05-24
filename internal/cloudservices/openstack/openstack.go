package openstack

import (
	"os"

	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

func WithLogger(logger *slog.Logger) OpenStackOption {
	return func(s *openStackService) {
		s.logger = logger
	}
}

func WithRegion(region string) OpenStackOption {
	return func(s *openStackService) {
		s.region = region
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

	return func(s *openStackService) {
		s.authOptions = authOptions
	}
}

func SyncDatabase(db *gorm.DB) error {
	err := db.AutoMigrate(&OpenStackProject{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&OpenStackOrganization{})
	if err != nil {
		return err
	}

	return nil
}

func NewOpenStackService(openStackOptions ...OpenStackOption) (types.CloudService, error) {
	s := openStackService{}

	for _, openStackOption := range openStackOptions {
		openStackOption(&s)
	}

	if s.logger == nil {
		s.logger = slog.New(slog.HandlerOptions{Level: slog.LevelInfo}.NewTextHandler(os.Stdout))
		s.logger.Info("no logger was provided, using default")
	}

	if s.region == "" {
		s.region = "sto1"
		s.logger.Debug("using default region", "region", s.region)
	}

	providerClient, err := openstack.AuthenticatedClient(*s.authOptions)
	if err != nil {
		return nil, err
	}

	s.providerClient = providerClient

	return &s, nil
}
