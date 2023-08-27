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
	"reflect"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestGetDeployment(t *testing.T) {
	tt := []struct {
		name         string
		deploymentID string
		deployments  []v1.Deployment
		expected     v1.Deployment
	}{
		{
			name:         "test single",
			deploymentID: "52b321cb-f9c5-43ba-bd35-ddc909ecfb64",
			deployments: []v1.Deployment{
				{
					ID: uuid.MustParse("52b321cb-f9c5-43ba-bd35-ddc909ecfb64"),
				},
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Deployment{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			h := handler{
				db:     db,
				logger: logger,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/deployments/:deploymentID", h.GetDeployment)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/deployments", tc.deploymentID),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}
		})
	}
}

func TestGetDeploymentErrors(t *testing.T) {
	tt := []struct {
		name         string
		deploymentID string
		deployments  []v1.Deployment
		expected     int
	}{
		{
			name:         "test missing",
			deploymentID: "c1e4b45e-cfe3-4fc7-a73a-2a3908524271",
			deployments: []v1.Deployment{
				{
					ID: uuid.MustParse("6c29ac51-2a27-4ab4-a030-77ebdddcf1c8"),
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
			db.AutoMigrate(&v1.Deployment{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}

			h := handler{
				db:     db,
				logger: logger,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/deployments/:deploymentID", h.GetDeployment)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/deployments", tc.deploymentID),
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

func TestGetClusterDeployments(t *testing.T) {
	tt := []struct {
		name               string
		clusterID          string
		deployments        []v1.Deployment
		clustermockOptions []clustermock.MockOption
		expected           []v1.Deployment
	}{
		{
			name:      "test single deployment",
			clusterID: "cluster-123",
			deployments: []v1.Deployment{
				{
					ID:        uuid.MustParse("115590c5-c5f5-48d3-95b4-5fd6a1d3e77f"),
					Name:      "test",
					ClusterID: "cluster-123",
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						ID:           "cluster-123",
						Name:         "test",
						Organization: "hello123",
					},
				}),
			},
			expected: []v1.Deployment{
				{
					ID:        uuid.MustParse("115590c5-c5f5-48d3-95b4-5fd6a1d3e77f"),
					Name:      "test",
					ClusterID: "cluster-123",
				},
			},
		},
		{
			name:      "test multiple deployments",
			clusterID: "cluster-123",
			deployments: []v1.Deployment{
				{
					ID:        uuid.MustParse("9f5be117-7a87-4b14-8788-42b595cd7679"),
					Name:      "test1",
					ClusterID: "cluster-123",
				},
				{
					ID:        uuid.MustParse("d40c37d3-7465-4bc6-bfbf-19669f05a16a"),
					Name:      "test2",
					ClusterID: "cluster-234",
				},
				{
					ID:        uuid.MustParse("a7743bee-d4cc-4342-b7bd-d149fa26f38f"),
					Name:      "test3",
					ClusterID: "cluster-123",
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						ID:           "cluster-123",
						Name:         "test",
						Organization: "hello123",
					},
				}),
			},
			expected: []v1.Deployment{
				{
					ID:        uuid.MustParse("9f5be117-7a87-4b14-8788-42b595cd7679"),
					Name:      "test1",
					ClusterID: "cluster-123",
				},
				{
					ID:        uuid.MustParse("a7743bee-d4cc-4342-b7bd-d149fa26f38f"),
					Name:      "test3",
					ClusterID: "cluster-123",
				},
			},
		},
		{
			name:      "test cluster without deployments",
			clusterID: "cluster-123",
			deployments: []v1.Deployment{
				{
					ID:        uuid.MustParse("b6cf669a-601f-4543-9a3c-d65da2d176d2"),
					Name:      "test1",
					ClusterID: "cluster-234",
				},
				{
					ID:        uuid.MustParse("1748bcf1-92c7-482e-a07c-a808701b2d84"),
					Name:      "test2",
					ClusterID: "cluster-345",
				},
				{
					ID:        uuid.MustParse("fd9786ad-6722-4ac4-9e18-6a128472eb60"),
					Name:      "test3",
					ClusterID: "cluster-456",
				},
			},
			clustermockOptions: []clustermock.MockOption{
				clustermock.WithClusters(map[string]v1.Cluster{
					"cluster-123": {
						ID:           "cluster-123",
						Name:         "test",
						Organization: "hello123",
					},
				}),
			},
			expected: []v1.Deployment{},
		},
	}

	gin.SetMode(gin.ReleaseMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Deployment{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			clusterService := clustermock.NewMockClusterService(tc.clustermockOptions...)

			h := handler{
				db:             db,
				logger:         logger,
				clusterService: clusterService,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.GET("/v1/clusters/:clusterID/deployments", h.GetClusterDeployments)

			c.Request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path: path.Join("/v1/clusters/", tc.clusterID, "deployments"),
				},
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			var actual []v1.Deployment
			err = json.Unmarshal(b, &actual)
			if err != nil {
				t.Fatalf("expected no error unmarshalling reponse, got %s", err)
			}

			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("cmp: %#v, %#v", actual, tc.expected)

			}
		})
	}
}

func TestDeleteDeployment(t *testing.T) {
	tt := []struct {
		name         string
		deploymentID string
		deployments  []v1.Deployment
	}{
		{
			name:         "test single",
			deploymentID: "33de82a0-4133-45dc-b319-ab6a8a1daebc",
			deployments: []v1.Deployment{
				{
					ID:   uuid.MustParse("33de82a0-4133-45dc-b319-ab6a8a1daebc"),
					Name: "test-123",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Deployment{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			h := handler{
				db:     db,
				logger: logger,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.DELETE("/v1/deployments/:deploymentID", h.DeleteDeployment)

			c.Request = &http.Request{
				Method: http.MethodDelete,
				URL: &url.URL{
					Path: path.Join("/v1/deployments", tc.deploymentID),
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

func TestPostClusterDeployments(t *testing.T) {
	tt := []struct {
		name       string
		clusterID  string
		deployment v1.Deployment
	}{
		{
			name:      "test helm",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				HelmChart: "test",
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
			db.AutoMigrate(&v1.Deployment{})

			h := handler{
				db:     db,
				logger: logger,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.POST("/v1/clusters/:clusterID/deployments", h.PostClusterDeployments)

			b, err := json.Marshal(tc.deployment)
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			req, err := http.NewRequest(http.MethodPost, u.String(), buf)

			r.ServeHTTP(w, req)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}
		})
	}
}

func TestPostClusterDeploymentsErrors(t *testing.T) {
	tt := []struct {
		name       string
		clusterID  string
		deployment v1.Deployment
		existing   []v1.Deployment
		expected   int
	}{
		{
			name:      "test invalid name",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				Name: "InvalidName",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test invalid container image",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				Name:           "test",
				ContainerImage: "http://localhost:1234/my-image",
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test name already in-use",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				Name: "test",
			},
			existing: []v1.Deployment{
				{
					Name:      "test",
					ClusterID: "cluster-123",
				},
			},
			expected: http.StatusConflict,
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
			db.AutoMigrate(&v1.Deployment{})

			for _, existing := range tc.existing {
				err := db.Create(&existing).Error
				if err != nil {
					t.Fatalf("error creating deployment in test database: %s", err)
				}
			}

			h := handler{
				db:     db,
				logger: logger,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.POST("/v1/clusters/:clusterID/deployments", h.PostClusterDeployments)

			b, err := json.Marshal(tc.deployment)
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			req, err := http.NewRequest(http.MethodPost, u.String(), buf)

			r.ServeHTTP(w, req)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}

func TestPostClusterDeploymentsContainerImage(t *testing.T) {
	tt := []struct {
		name       string
		clusterID  string
		deployment v1.Deployment
	}{
		{
			name:      "test container image",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				ContainerImage: "test",
			},
		},
		{
			name:      "test port",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				ContainerImage: "nginx:l.2",
				Port:           1234,
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
			db.AutoMigrate(&v1.Deployment{})

			dirTemp, err := os.MkdirTemp("", "dockyards-")
			if err != nil {
				t.Fatalf("error creating temporary directory: %s", err)
			}

			h := handler{
				db:             db,
				logger:         logger,
				gitProjectRoot: dirTemp,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.POST("/v1/clusters/:clusterID/deployments", h.PostClusterDeployments)

			b, err := json.Marshal(tc.deployment)
			buf := bytes.NewBuffer(b)

			u := url.URL{
				Path: path.Join("/v1/clusters", tc.clusterID, "deployments"),
			}

			req, err := http.NewRequest(http.MethodPost, u.String(), buf)

			r.ServeHTTP(w, req)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
			}
		})
	}
}
