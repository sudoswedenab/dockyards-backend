package ipam

import (
	"log/slog"
	"net/netip"

	"gorm.io/gorm"
)

type IPManager interface {
	AllocateAddr(netip.Prefix, string) (netip.Addr, error)
	ReleaseAddr(netip.Addr) error
}

type ipManager struct {
	logger *slog.Logger
	db     *gorm.DB
}

var _ IPManager = &ipManager{}

type ManagerOption func(*ipManager)

func WithLogger(logger *slog.Logger) ManagerOption {
	return func(m *ipManager) {
		m.logger = logger
	}
}

func WithDB(db *gorm.DB) ManagerOption {
	return func(m *ipManager) {
		m.db = db
	}
}

func NewIPManager(managerOptions ...ManagerOption) (*ipManager, error) {
	m := ipManager{}

	for _, managerOption := range managerOptions {
		managerOption(&m)
	}

	err := m.db.AutoMigrate(&ipamAllocation{})
	if err != nil {
		return nil, err
	}

	return &m, nil
}
