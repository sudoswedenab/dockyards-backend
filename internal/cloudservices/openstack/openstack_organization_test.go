package openstack

import (
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

func TestOpenStackCreateOrganization(t *testing.T) {
	tt := []struct {
		name         string
		organization model.Organization
		projects     []OpenStackProject
		expected     string
	}{
		{
			name: "test simple",
			organization: model.Organization{
				ID:   uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				Name: "simple",
			},
			projects: []OpenStackProject{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					OpenStackID: "abc123",
				},
			},
			expected: "abc123",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating database: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("unexpected error preparing database: %s", err)
			}
			err = SyncDatabase(db)
			if err != nil {
				t.Fatalf("unexpected error preparing database: %s", err)
			}
			err = db.Create(&tc.organization).Error
			if err != nil {
				t.Fatalf("unexpected error creating organization in database: %s", err)
			}
			for _, project := range tc.projects {
				err := db.Create(&project).Error
				if err != nil {
					t.Errorf("unexpected error creating projects in database: %s", err)
				}
			}

			logger := slog.New(slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stdout))
			s := openStackService{
				db:     db,
				logger: logger,
			}
			actual, err := s.CreateOrganization(&tc.organization)
			if err != nil {
				t.Fatalf("unexpected error creating organization: %s", err)
			}
			if actual != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, actual)
			}
		})
	}
}

func TestOpenStackGetOrganization(t *testing.T) {
	tt := []struct {
		name                   string
		organization           model.Organization
		openStackProjects      []OpenStackProject
		openStackOrganizations []OpenStackOrganization
		expected               string
	}{
		{
			name: "test single",
			organization: model.Organization{
				ID:   uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				Name: "single",
			},
			openStackProjects: []OpenStackProject{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					OpenStackID: "abc123",
				},
			},
			openStackOrganizations: []OpenStackOrganization{
				{
					ID:                 uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
					OpenStackProjectID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					OrganizationID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
			},
			expected: "abc123",
		},
		{
			name: "test multiple projects",
			organization: model.Organization{
				ID:   uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				Name: "multiple",
			},
			openStackProjects: []OpenStackProject{
				{
					ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					OpenStackID: "abc123",
				},
				{
					ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					OpenStackID: "abc234",
				},
				{
					ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
					OpenStackID: "abc345",
				},
				{
					ID:          uuid.MustParse("44444444-4444-4444-4444-444444444444"),
					OpenStackID: "abc456",
				},
				{
					ID:          uuid.MustParse("55555555-5555-5555-5555-555555555555"),
					OpenStackID: "abc567",
				},
			},
			openStackOrganizations: []OpenStackOrganization{
				{
					ID:                 uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffff1"),
					OpenStackProjectID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					OrganizationID:     uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				{
					ID:                 uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffff2"),
					OpenStackProjectID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
					OrganizationID:     uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				},
				{
					ID:                 uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffff3"),
					OpenStackProjectID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
					OrganizationID:     uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
			},
			expected: "abc345",
		},
	}

	logger := slog.New(slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stdout))

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating database: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("unexpected error preparing database: %s", err)
			}
			err = SyncDatabase(db)
			if err != nil {
				t.Fatalf("unexpected error preparing database: %s", err)
			}
			for _, openStackProject := range tc.openStackProjects {
				err := db.Create(&openStackProject).Error
				if err != nil {
					t.Fatalf("unexpected error creating openstack project in database: %s", err)
				}
			}
			for _, openStackOrganization := range tc.openStackOrganizations {
				err := db.Create(&openStackOrganization).Error
				if err != nil {
					t.Fatalf("unexpected error creating openstack organization in database: %s", err)
				}
			}

			s := &openStackService{logger: logger, db: db}
			actual, err := s.GetOrganization(&tc.organization)
			if err != nil {
				t.Fatalf("unexpected error getting organization: %s", err)
			}
			if actual != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, actual)
			}
		})
	}
}
