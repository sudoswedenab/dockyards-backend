package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestGetNodePool(t *testing.T) {
	tt := []struct {
		name               string
		nodePoolID         string
		clustermockOptions []clustermock.MockOption
		expected           v1.NodePool
	}{
		{
			name:       "test single node",
			nodePoolID: "node-pool-123",
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithNodePools(map[string]v1.NodePool{
					"node-pool-123": {
						Name: "test-pool",
						Nodes: []v1.Node{
							{
								ID:   "node-123",
								Name: "node-pool-123-1",
							},
						},
					},
				}),
			},
			expected: v1.NodePool{
				Name: "test-pool",
				Nodes: []v1.Node{
					{
						ID:   "node-123",
						Name: "node-pool-123-1",
					},
				},
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

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				db:             db,
				logger:         logger,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/node-pools/:nodePoolID", h.GetNodePool)

			c.Params = []gin.Param{
				{Key: "nodePoolID", Value: tc.nodePoolID},
			}
			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/node-pools", tc.nodePoolID),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual v1.NodePool
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body to json: %s", err)
			}

			if !cmp.Equal(tc.expected, actual) {
				t.Errorf("diff: %s", cmp.Diff(actual, tc.expected))
			}
		})
	}
}

func TestPostClusterNodePools(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		nodePoolOptions    v1.NodePoolOptions
		clustermockOptions []clustermock.MockOption
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		expected           v1.NodePool
	}{
		{
			name:      "test simple",
			clusterID: "cluster-123",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name:         "cluster-123",
						Organization: "test-org",
					},
				}),
			},
			user: v1.User{
				ID: uuid.MustParse("d80ff784-20fe-4bcc-b52f-e57764111c9a"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("d80ff784-20fe-4bcc-b52f-e57764111c9a"),
				},
			},
			organizations: []v1.Organization{
				{
					ID:   uuid.MustParse("3928f445-d53c-4a23-9663-77382a361d17"),
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("d80ff784-20fe-4bcc-b52f-e57764111c9a"),
						},
					},
				},
			},
			expected: v1.NodePool{
				Name: "test",
			},
		},
		{
			name:      "test complex",
			clusterID: "cluster-123",
			nodePoolOptions: v1.NodePoolOptions{
				Name:                       "test2",
				Quantity:                   3,
				LoadBalancer:               util.Ptr(true),
				ControlPlaneComponentsOnly: util.Ptr(true),
				RAMSizeMb:                  util.Ptr(1234),
				CPUCount:                   util.Ptr(12),
				DiskSizeGb:                 util.Ptr(123),
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name:         "cluster-123",
						Organization: "test-org",
					},
				}),
			},
			user: v1.User{
				ID: uuid.MustParse("940b43ee-39d3-4ecb-a6bd-be25245d7eca"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("940b43ee-39d3-4ecb-a6bd-be25245d7eca"),
				},
			},
			organizations: []v1.Organization{
				{
					ID:   uuid.MustParse("a86dd064-4fa5-489f-ab29-6f49f92a38eb"),
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("940b43ee-39d3-4ecb-a6bd-be25245d7eca"),
						},
					},
				},
			},
			expected: v1.NodePool{
				Name:                       "test2",
				Quantity:                   3,
				LoadBalancer:               util.Ptr(true),
				ControlPlaneComponentsOnly: util.Ptr(true),
				RAMSizeMb:                  1234,
				CPUCount:                   12,
				DiskSizeGb:                 123,
			},
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
			db.AutoMigrate(v1.Organization{})
			for _, organization := range tc.organizations {
				err := db.Create(&organization).Error
				if err != nil {
					t.Fatalf("unexpected error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				db:             db,
				logger:         logger,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("user", tc.user)
			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}

			b, err := json.Marshal(tc.nodePoolOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test options: %s", err)
			}
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "node-pools"),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), buf)

			h.PostClusterNodePools(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}

			b, err = io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("unexpected error reading result body: %s", err)
			}

			var actual v1.NodePool
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body to json: %s", err)
			}

			if !cmp.Equal(tc.expected, actual) {
				t.Errorf("diff: %s", cmp.Diff(actual, tc.expected))
			}

		})
	}
}

func TestPostClusterNodePoolsErrors(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		nodePoolOptions    v1.NodePoolOptions
		clustermockOptions []clustermock.MockOption
		user               v1.User
		users              []v1.User
		organizations      []v1.Organization
		expected           int
	}{
		{
			name:      "test invalid cluster",
			clusterID: "cluster-234",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name: "cluster-123",
					},
				}),
			},
			expected: http.StatusUnauthorized,
		},
		{
			name:      "test invalid name",
			clusterID: "cluster-123",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "InvalidName",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test conflict name",
			clusterID: "cluster-123",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name:         "cluster-123",
						Organization: "test-org",
						NodePools: []v1.NodePool{
							{

								Name: "test",
							},
						},
					},
				}),
			},
			user: v1.User{
				ID: uuid.MustParse("df24c8f4-27f3-485a-ae7a-92546b3fb925"),
			},
			users: []v1.User{
				{
					ID: uuid.MustParse("df24c8f4-27f3-485a-ae7a-92546b3fb925"),
				},
			},
			organizations: []v1.Organization{
				{
					ID:   uuid.MustParse("ae19c385-6254-4d73-a2fa-53c29796ee91"),
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("df24c8f4-27f3-485a-ae7a-92546b3fb925"),
						}},
				},
			},
			expected: http.StatusConflict,
		},
		{
			name:      "test invalid membership",
			clusterID: "cluster-123",
			nodePoolOptions: v1.NodePoolOptions{
				Name: "test",
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						Name:         "cluster-123",
						Organization: "test-org",
					},
				}),
			},
			user: v1.User{
				ID: uuid.MustParse("44946295-97bc-4c24-8887-69d3f0ca0dad"),
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("44946295-97bc-4c24-8887-69d3f0ca0dad"),
					Name:  "user1",
					Email: "user1@dockyards.dev",
				},
				{
					ID:    uuid.MustParse("bbc144d1-0f5f-4f8b-8b8b-54d0619395bc"),
					Name:  "user2",
					Email: "user2@dockyards.dev",
				},
			},
			organizations: []v1.Organization{
				{
					ID:   uuid.MustParse("d3570450-a7e1-4201-a16f-b913ad6c7f11"),
					Name: "test-org",
					Users: []v1.User{
						{
							ID: uuid.MustParse("bbc144d1-0f5f-4f8b-8b8b-54d0619395bc"),
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
			db.AutoMigrate(v1.User{})
			db.AutoMigrate(v1.Organization{})
			for _, user := range tc.users {
				err := db.Create(&user).Error
				if err != nil {
					t.Fatalf("unexpected error creating user in test database: %s", err)
				}
			}
			for _, organization := range tc.organizations {
				err := db.Create(&organization).Error
				if err != nil {
					t.Fatalf("unexpected error creating organization in test database: %s", err)
				}
			}

			h := handler{
				clusterService: clustermock.NewMockClusterService(tc.clustermockOptions...),
				db:             db,
				logger:         logger,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("user", tc.user)
			c.Params = []gin.Param{
				{Key: "clusterID", Value: tc.clusterID},
			}

			b, err := json.Marshal(tc.nodePoolOptions)
			if err != nil {
				t.Fatalf("unexpected error marshalling test options: %s", err)
			}
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "node-pools"),
			}

			c.Request, err = http.NewRequest(http.MethodPost, u.String(), buf)

			h.PostClusterNodePools(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}
