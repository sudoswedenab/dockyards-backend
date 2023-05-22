package openstack

import (
	"errors"
	"os"

	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

func WithAuthInfo(authURL, applicationCredentialID, applicationCredentialSecret string) OpenStackOption {
	return func(s *openStackService) {
		s.authInfo = &clientconfig.AuthInfo{
			AuthURL:                     authURL,
			ApplicationCredentialID:     applicationCredentialID,
			ApplicationCredentialSecret: applicationCredentialSecret,
			AllowReauth:                 true,
		}
	}
}

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

	if s.authInfo == nil {
		return nil, errors.New("no auth information provided")
	}

	if s.logger == nil {
		s.logger = slog.New(slog.HandlerOptions{Level: slog.LevelInfo}.NewTextHandler(os.Stdout))
		s.logger.Info("no logger was provided, using default")
	}

	if s.region == "" {
		s.region = "sto1"
		s.logger.Debug("using default region", "region", s.region)
	}

	clientOpts := clientconfig.ClientOpts{
		AuthType: clientconfig.AuthV3ApplicationCredential,
		AuthInfo: s.authInfo,
	}

	providerClient, err := clientconfig.AuthenticatedClient(&clientOpts)
	if err != nil {
		return nil, err
	}

	s.providerClient = providerClient

	return &s, nil
}
