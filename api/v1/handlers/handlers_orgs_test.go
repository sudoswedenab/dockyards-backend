package handlers

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/cloudmock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestDeleteOrganization(t *testing.T) {
	tt := []struct {
		name               string
		organizationID     string
		organizations      []v1.Organization
		cloudmockOptions   []cloudmock.MockOption
		clustermockOptions []clustermock.MockOption
	}{
		{
			name:           "test simple",
			organizationID: "8f15221d-2e4c-4d5e-9288-3052a952ac4f",
			organizations: []v1.Organization{
				{
					ID:   uuid.MustParse("8f15221d-2e4c-4d5e-9288-3052a952ac4f"),
					Name: "test",
				},
			},
			cloudmockOptions: []cloudmock.MockOption{
				cloudmock.WithOrganizations(map[string]bool{
					"test": true,
				}),
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{}),
			},
		},
	}

	gin.SetMode(gin.ReleaseMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Organization{})

			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			cloudService := cloudmock.NewMockCloudService(tc.cloudmockOptions...)
			clusterService := clustermock.NewMockClusterService(tc.clustermockOptions...)

			h := handler{
				db:             db,
				logger:         logger,
				cloudService:   cloudService,
				clusterService: clusterService,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/orgs/:org", h.DeleteOrganization)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/orgs/", tc.organizationID),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusNoContent {
				t.Fatalf("expected status code %d, got %d", http.StatusNoContent, statusCode)
			}

		})
	}
}

func TestDeleteOrganizationErrors(t *testing.T) {
	tt := []struct {
		name               string
		organizationID     string
		organizations      []v1.Organization
		cloudmockOptions   []cloudmock.MockOption
		clustermockOptions []clustermock.MockOption
		expected           int
	}{
		{
			name:           "test missing organization",
			organizationID: "d14d89fb-246a-41e4-b73c-edb00924230f",
			organizations:  []v1.Organization{},
			expected:       http.StatusUnauthorized,
		},
		{
			name:           "test organization with clusters",
			organizationID: "8cbe7a7f-6967-4347-b58e-fd2e5937a563",
			organizations: []v1.Organization{
				{
					ID:   uuid.MustParse("8cbe7a7f-6967-4347-b58e-fd2e5937a563"),
					Name: "still-has-clusters",
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"still-has-clusters": v1.Cluster{
						Name:         "test-123",
						Organization: "still-has-clusters",
					},
				}),
			},
			expected: http.StatusForbidden,
		},
		{
			name:           "test organization missing in cloud service",
			organizationID: "0f479212-3bc3-4219-808e-c327c8e22390",
			organizations: []v1.Organization{
				{
					ID: uuid.MustParse("0f479212-3bc3-4219-808e-c327c8e22390"),
				},
			},
			expected: http.StatusInternalServerError,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Organization{})

			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			cloudService := cloudmock.NewMockCloudService(tc.cloudmockOptions...)
			clusterService := clustermock.NewMockClusterService(tc.clustermockOptions...)

			h := handler{
				db:             db,
				logger:         logger,
				cloudService:   cloudService,
				clusterService: clusterService,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/orgs/:org", h.DeleteOrganization)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/orgs/", tc.organizationID),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}

}
