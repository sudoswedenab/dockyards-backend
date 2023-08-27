package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/cloudmock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestPostOrgClusters(t *testing.T) {
	tt := []struct {
		name             string
		organizationName string
		user             v1.User
		users            []v1.User
		organizations    []v1.Organization
		clusterOptions   v1.ClusterOptions
	}{
		{
			name:             "test recommended",
			organizationName: "test-org",
			user: v1.User{
				ID: uuid.MustParse("fec813fc-7938-4cb9-ba12-bb28f6b1f5d9"),
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("fec813fc-7938-4cb9-ba12-bb28f6b1f5d9"),
					Email: "test@dockyards.dev",
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("fec813fc-7938-4cb9-ba12-bb28f6b1f5d9"),
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test",
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("error syncing database")
			}
			for _, u := range tc.users {
				err := db.Create(&u).Error
				if err != nil {
					t.Fatalf("error creating user in test database: %s", err)
				}
			}
			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(),
				cloudService:   cloudmock.NewMockCloudService(),
				logger:         logger,
				db:             db,
			}

			b, err := json.Marshal(tc.clusterOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test cluster options: %s", err)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "org", Value: tc.organizationName},
			}
			c.Set("user", tc.user)

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationName, "clusters"),
			}
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.PostOrgClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}
		})
	}
}

func TestPostOrgClustersErrors(t *testing.T) {
	tt := []struct {
		name               string
		organizationName   string
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		clusterOptions     v1.ClusterOptions
		clustermockOptions []clustermock.MockOption
		expected           int
	}{
		{
			name:             "test invalid organization",
			organizationName: "test-org",
			expected:         http.StatusUnauthorized,
		},
		{
			name:             "test invalid cluster name",
			organizationName: "test-org",
			user: v1.User{
				ID:    uuid.MustParse("82aaf116-666f-4846-9e10-defa79a4df3d"),
				Email: "test@dockyards.dev",
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("82aaf116-666f-4846-9e10-defa79a4df3d"),
					Email: "test@dockyards.dev",
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID:    uuid.MustParse("82aaf116-666f-4846-9e10-defa79a4df3d"),
							Email: "test@dockyards.dev",
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "InvalidClusterName",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:             "test invalid node pool name",
			organizationName: "test-org",
			user: v1.User{
				ID: uuid.MustParse("e7282b48-f8b6-4042-8f4c-12ec59fe3a87"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("e7282b48-f8b6-4042-8f4c-12ec59fe3a87"),
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("e7282b48-f8b6-4042-8f4c-12ec59fe3a87"),
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test-cluster",
				NodePoolOptions: util.Ptr([]v1.NodePoolOptions{
					{
						Name: "InvalidNodePoolName",
					},
				}),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:             "test invalid membership",
			organizationName: "test-org",
			user: v1.User{
				ID: uuid.MustParse("62034914-3f46-4c71-810f-14ab985399bc"),
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("62034914-3f46-4c71-810f-14ab985399bc"),
					Name:  "user1",
					Email: "user1@dockyards.dev",
				},
				{
					ID:    uuid.MustParse("af510e3e-e667-4500-8a73-12f2163f849e"),
					Name:  "user2",
					Email: "user2@dockyards.dev",
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("af510e3e-e667-4500-8a73-12f2163f849e"),
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:             "test existing cluster name",
			organizationName: "test-org",
			user: v1.User{
				ID: uuid.MustParse("c185f9d3-b4c4-4cb1-a567-f786c9ac4a2f"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("c185f9d3-b4c4-4cb1-a567-f786c9ac4a2f"),
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("c185f9d3-b4c4-4cb1-a567-f786c9ac4a2f"),
						},
					},
				},
			},
			clusterOptions: v1.ClusterOptions{
				Name: "test-cluster",
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"test-cluster": {
						Name:         "test-cluster",
						Organization: "test-org",
					},
				}),
			},
			expected: http.StatusConflict,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("error syncing database")
			}
			for _, u := range tc.users {
				err := db.Create(&u).Error
				if err != nil {
					t.Fatalf("error creating user in test database: %s", err)
				}
			}
			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				cloudService:   cloudmock.NewMockCloudService(),
				logger:         logger,
				db:             db,
			}

			b, err := json.Marshal(tc.clusterOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test cluster options: %s", err)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "org", Value: tc.organizationName},
			}
			c.Set("user", tc.user)

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationName, "clusters"),
			}
			c.Request, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.PostOrgClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestDeleteOrgClusters(t *testing.T) {
	tt := []struct {
		name               string
		organizationName   string
		clusterName        string
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		clustermockOptions []clustermock.MockOption
	}{
		{
			name:             "test simple",
			organizationName: "test-org",
			clusterName:      "test-cluster",
			user: v1.User{
				ID: uuid.MustParse("7994b631-399a-41e6-9c6c-200391f8f87d"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("7994b631-399a-41e6-9c6c-200391f8f87d"),
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("7994b631-399a-41e6-9c6c-200391f8f87d"),
						},
					},
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"test-cluster": {
						Organization: "test-org",
						Name:         "test-cluster",
					},
				}),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("error syncing database")
			}
			for _, u := range tc.users {
				err := db.Create(&u).Error
				if err != nil {
					t.Fatalf("error creating user in test database: %s", err)
				}
			}
			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				cloudService:   cloudmock.NewMockCloudService(),
				logger:         logger,
				db:             db,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "org", Value: tc.organizationName},
				{Key: "cluster", Value: tc.clusterName},
			}
			c.Set("user", tc.user)

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationName, "clusters", tc.clusterName),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.DeleteOrgClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusAccepted {
				t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
			}
		})
	}

}

func TestDeleteOrgClustersErrors(t *testing.T) {
	tt := []struct {
		name               string
		organizationName   string
		clusterName        string
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		clustermockOptions []clustermock.MockOption
		expected           int
	}{
		{
			name:     "test empty",
			expected: http.StatusBadRequest,
		},
		{
			name:             "test empty cluster name",
			organizationName: "test-org",
			expected:         http.StatusBadRequest,
		},
		{
			name:             "test invalid organization",
			organizationName: "test-org",
			clusterName:      "test-cluster",
			user: v1.User{
				ID: uuid.MustParse("e5cd33c8-cddf-494f-9156-3ddc82b7c2f5"),
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:             "test invalid cluster",
			organizationName: "test-org",
			clusterName:      "test-cluster",
			user: v1.User{
				ID: uuid.MustParse("f5cf8f91-2b38-4bf4-bb52-d4d4f79f42c3"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("f5cf8f91-2b38-4bf4-bb52-d4d4f79f42c3"),
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("f5cf8f91-2b38-4bf4-bb52-d4d4f79f42c3"),
						},
					},
				},
			},
			expected: http.StatusInternalServerError,
		},
		{
			name:             "test invalid organization membership",
			organizationName: "test-org",
			clusterName:      "test-cluster",
			user: v1.User{
				ID: uuid.MustParse("8ce52ca1-1931-49a1-8ddf-62bf3870a4bf"),
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("8ce52ca1-1931-49a1-8ddf-62bf3870a4bf"),
					Name:  "user1",
					Email: "user1@dockyards.dev",
				},
				{
					ID:    uuid.MustParse("0b8f6617-eba7-4360-b73a-11dac2286a40"),
					Name:  "user2",
					Email: "user2@dockyards.dev",
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("0b8f6617-eba7-4360-b73a-11dac2286a40"),
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("error syncing database")
			}
			for _, u := range tc.users {
				err := db.Create(&u).Error
				if err != nil {
					t.Fatalf("error creating user in test database: %s", err)
				}
			}
			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				cloudService:   cloudmock.NewMockCloudService(),
				logger:         logger,
				db:             db,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "org", Value: tc.organizationName},
				{Key: "cluster", Value: tc.clusterName},
			}
			c.Set("user", tc.user)

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationName, "clusters", tc.clusterName),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.DeleteOrgClusters(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestGetCluster(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		clustermockOptions []clustermock.MockOption
	}{
		{
			name:      "test simple",
			clusterID: "cluster-123",
			user: v1.User{
				ID: uuid.MustParse("f235721e-8e34-4b57-a6aa-8f6d31162a41"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("f235721e-8e34-4b57-a6aa-8f6d31162a41"),
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("f235721e-8e34-4b57-a6aa-8f6d31162a41"),
						},
					},
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name:         "cluster-123",
						Organization: "test-org",
					},
				}),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("error syncing database")
			}
			for _, u := range tc.users {
				err := db.Create(&u).Error
				if err != nil {
					t.Fatalf("error creating user in test database: %s", err)
				}
			}
			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				cloudService:   cloudmock.NewMockCloudService(),
				logger:         logger,
				db:             db,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("user", tc.user)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetCluster(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}
		})
	}
}

func TestGetClusterErrors(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		clustermockOptions []clustermock.MockOption
		expected           int
	}{
		{
			name:     "test empty",
			expected: http.StatusBadRequest,
		},
		{
			name:      "test invalid cluster",
			clusterID: "cluster-123",
			expected:  http.StatusUnauthorized,
		},
		{
			name:      "test invalid membership",
			clusterID: "cluster-123",
			user: v1.User{
				ID: uuid.MustParse("f6f6531f-ab6c-4237-b1cb-76133674465f"),
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("f6f6531f-ab6c-4237-b1cb-76133674465f"),
					Name:  "user1",
					Email: "user1@dockyards.dev",
				},
				{
					ID:    uuid.MustParse("afb03005-d51d-4387-9857-83125ff505d5"),
					Name:  "user2",
					Email: "user2@dockyards.dev",
				},
			},
			organizations: []v1.Organization{
				{
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("afb03005-d51d-4387-9857-83125ff505d5"),
						},
					},
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name:         "cluster-123",
						Organization: "test-org",
					},
				}),
			},
			expected: http.StatusUnauthorized,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			err = internal.SyncDataBase(db)
			if err != nil {
				t.Fatalf("error syncing database")
			}
			for _, u := range tc.users {
				err := db.Create(&u).Error
				if err != nil {
					t.Fatalf("error creating user in test database: %s", err)
				}
			}
			for _, o := range tc.organizations {
				err := db.Create(&o).Error
				if err != nil {
					t.Fatalf("error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				cloudService:   cloudmock.NewMockCloudService(),
				logger:         logger,
				db:             db,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}
			c.Set("user", tc.user)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing test request: %s", err)
			}

			h.GetCluster(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}
