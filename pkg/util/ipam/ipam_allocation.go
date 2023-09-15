package ipam

import (
	"errors"
	"net/netip"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrPrefixFull       = errors.New("prefix is full")
	ErrAddrNotAllocated = errors.New("address not allocated")
)

type ipamAllocation struct {
	ID   uuid.UUID  `gorm:"id"`
	Addr netip.Addr `gorm:"unique;serializer:json"`
	Tag  string     `gorm:"tag"`
}

func (m *ipManager) AllocateAddr(prefix netip.Prefix, tag string) (netip.Addr, error) {
	addr := prefix.Addr()

	for prefix.Contains(addr) {
		var allocation ipamAllocation

		err := m.db.Where(ipamAllocation{Addr: addr}).Take(&allocation).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return netip.Addr{}, err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			allocation := ipamAllocation{
				ID:   uuid.New(),
				Addr: addr,
				Tag:  tag,
			}

			err = m.db.Create(&allocation).Error
			if err != nil {
				return netip.Addr{}, err
			}

			return addr, nil
		}

		addr = addr.Next()
	}

	return prefix.Addr(), ErrPrefixFull
}

func (m *ipManager) ReleaseAddr(addr netip.Addr) error {
	allocation := ipamAllocation{
		Addr: addr,
	}

	err := m.db.Where(allocation).Take(&allocation).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAddrNotAllocated
	}

	m.logger.Debug("release allocation", "id", allocation.ID)

	err = m.db.Delete(&allocation).Error
	if err != nil {
		return err
	}

	return nil
}
