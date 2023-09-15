package ipam

import (
	"errors"
	"log/slog"
	"net/netip"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestAllocateAddr(t *testing.T) {
	tt := []struct {
		name        string
		prefix      netip.Prefix
		tag         string
		allocations []ipamAllocation
		expected    netip.Addr
	}{
		{
			name:     "test simple",
			prefix:   netip.MustParsePrefix("1.2.3.4/30"),
			expected: netip.MustParseAddr("1.2.3.4"),
		},
		{
			name:   "test with allocations",
			prefix: netip.MustParsePrefix("1.2.3.4/30"),
			allocations: []ipamAllocation{
				{
					ID:   uuid.MustParse("56271cfc-d3ee-4e11-9882-a60d455ae905"),
					Addr: netip.MustParseAddr("1.2.3.4"),
				},
				{
					ID:   uuid.MustParse("1cb349d4-8a0e-4d56-9eac-ce830f1564b0"),
					Addr: netip.MustParseAddr("1.2.3.5"),
				},
			},
			expected: netip.MustParseAddr("1.2.3.6"),
		},
		{
			name:   "test allocations with holes",
			prefix: netip.MustParsePrefix("1.2.3.4/30"),
			allocations: []ipamAllocation{
				{
					ID:   uuid.MustParse("e37958d6-f31d-4110-bfd4-14f146bfe986"),
					Addr: netip.MustParseAddr("1.2.3.4"),
				},
				{
					ID:   uuid.MustParse("260ae059-876c-4f83-b0d2-19d8c95a8878"),
					Addr: netip.MustParseAddr("1.2.3.6"),
				},
			},
			expected: netip.MustParseAddr("1.2.3.5"),
		},
		{
			name:     "test with tag",
			prefix:   netip.MustParsePrefix("1.2.3.4/32"),
			tag:      "test",
			expected: netip.MustParseAddr("1.2.3.4"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&ipamAllocation{})
			for _, allocation := range tc.allocations {
				err := db.Create(&allocation).Error
				if err != nil {
					t.Fatalf("unexpected error creating allocation in test database: %s", err)
				}
			}

			m := ipManager{
				logger: logger,
				db:     db,
			}
			actual, err := m.AllocateAddr(tc.prefix, tc.tag)
			if err != nil {
				t.Fatalf("error allocating address from prefix: %s", err)
			}

			if actual != tc.expected {
				t.Errorf("expected addr %s, got %s", tc.expected, actual)
			}
		})
	}
}

func TestAllocateIPErrors(t *testing.T) {
	tt := []struct {
		name        string
		prefix      netip.Prefix
		tag         string
		allocations []ipamAllocation
		expected    error
	}{
		{
			name:   "test full prefix",
			prefix: netip.MustParsePrefix("1.2.3.4/31"),
			allocations: []ipamAllocation{
				{
					ID:   uuid.MustParse("a5b083f7-b761-4135-80c6-50600304f707"),
					Addr: netip.MustParseAddr("1.2.3.4"),
				},
				{
					ID:   uuid.MustParse("83ccc59b-d35f-4c65-8019-2b63ae66e59e"),
					Addr: netip.MustParseAddr("1.2.3.5"),
				},
			},
			expected: ErrPrefixFull,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&ipamAllocation{})
			for _, allocation := range tc.allocations {
				err := db.Create(&allocation).Error
				if err != nil {
					t.Fatalf("unexpected error creating allocation in test database: %s", err)
				}
			}

			m := ipManager{
				logger: logger,
				db:     db,
			}
			_, err = m.AllocateAddr(tc.prefix, tc.tag)
			if !errors.Is(err, tc.expected) {
				t.Errorf("expected error '%s', got '%s'", tc.expected, err)
			}
		})
	}
}

func TestReleaseAddr(t *testing.T) {
	tt := []struct {
		name        string
		allocations []ipamAllocation
		addr        netip.Addr
	}{
		{
			name: "test simple",
			allocations: []ipamAllocation{
				{
					ID:   uuid.MustParse("0d3a881b-f83e-4b5d-82f0-825b94fac0e0"),
					Addr: netip.MustParseAddr("1.2.3.4"),
				},
			},
			addr: netip.MustParseAddr("1.2.3.4"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&ipamAllocation{})
			for _, allocation := range tc.allocations {
				err := db.Create(&allocation).Error
				if err != nil {
					t.Fatalf("unexpected error creating allocation in test database: %s", err)
				}
			}

			m := ipManager{
				logger: logger,
				db:     db,
			}

			err = m.ReleaseAddr(tc.addr)
			if err != nil {
				t.Errorf("expected error to be nil, got %s", err)
			}
		})
	}
}

func TestReleaseAddrErrors(t *testing.T) {
	tt := []struct {
		name        string
		allocations []ipamAllocation
		addr        netip.Addr
		expected    error
	}{
		{
			name:     "test unallocated address",
			addr:     netip.MustParseAddr("1.2.3.4"),
			expected: ErrAddrNotAllocated,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&ipamAllocation{})
			for _, allocation := range tc.allocations {
				err := db.Create(&allocation).Error
				if err != nil {
					t.Fatalf("unexpected error creating allocation in test database: %s", err)
				}
			}

			m := ipManager{
				logger: logger,
				db:     db,
			}

			err = m.ReleaseAddr(tc.addr)
			if !errors.Is(err, tc.expected) {
				t.Errorf("expected error '%s', got '%s'", tc.expected, err)
			}
		})
	}
}
