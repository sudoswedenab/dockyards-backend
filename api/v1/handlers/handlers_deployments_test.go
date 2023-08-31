package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

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

func TestGetDeployment(t *testing.T) {
	now := time.Now()

	tt := []struct {
		name               string
		deploymentID       string
		deployments        []v1.Deployment
		deploymentStatuses []v1.DeploymentStatus
		expected           v1.Deployment
	}{
		{
			name:         "test single",
			deploymentID: "52b321cb-f9c5-43ba-bd35-ddc909ecfb64",
			deployments: []v1.Deployment{
				{
					ID: uuid.MustParse("52b321cb-f9c5-43ba-bd35-ddc909ecfb64"),
				},
			},
			expected: v1.Deployment{
				ID: uuid.MustParse("52b321cb-f9c5-43ba-bd35-ddc909ecfb64"),
			},
		},
		{
			name:         "test complex",
			deploymentID: "9f72e4e6-412c-47a9-b3e8-8704e129db57",
			deployments: []v1.Deployment{
				{
					ID:             uuid.MustParse("9f72e4e6-412c-47a9-b3e8-8704e129db57"),
					Name:           util.Ptr("test"),
					ContainerImage: util.Ptr("docker.io/library/nginx:latest"),
					Port:           util.Ptr(1234),
				},
			},
			expected: v1.Deployment{
				ID:             uuid.MustParse("9f72e4e6-412c-47a9-b3e8-8704e129db57"),
				Name:           util.Ptr("test"),
				ContainerImage: util.Ptr("docker.io/library/nginx:latest"),
				Port:           util.Ptr(1234),
			},
		},
		{
			name:         "test deployment with single status",
			deploymentID: "63f4b165-d9e4-4653-a2a4-92b14ff6153e",
			deployments: []v1.Deployment{
				{
					ID: uuid.MustParse("63f4b165-d9e4-4653-a2a4-92b14ff6153e"),
				},
			},
			deploymentStatuses: []v1.DeploymentStatus{
				{
					ID:           uuid.MustParse("5024648b-0222-4b6a-9845-26d051c2613c"),
					DeploymentID: uuid.MustParse("63f4b165-d9e4-4653-a2a4-92b14ff6153e"),
					CreatedAt:    now,
					State:        util.Ptr("testing"),
					Health:       util.Ptr(v1.DeploymentStatusHealthWarning),
				},
			},
			expected: v1.Deployment{
				ID: uuid.MustParse("63f4b165-d9e4-4653-a2a4-92b14ff6153e"),
				Status: v1.DeploymentStatus{
					ID:           uuid.MustParse("5024648b-0222-4b6a-9845-26d051c2613c"),
					DeploymentID: uuid.MustParse("63f4b165-d9e4-4653-a2a4-92b14ff6153e"),
					CreatedAt:    now,
					State:        util.Ptr("testing"),
					Health:       util.Ptr(v1.DeploymentStatusHealthWarning),
				},
			},
		},
		{
			name:         "test deployment with multiple statuses",
			deploymentID: "f658aec8-0361-4f6c-ab10-1959ad433156",
			deployments: []v1.Deployment{
				{
					ID: uuid.MustParse("f658aec8-0361-4f6c-ab10-1959ad433156"),
				},
			},
			deploymentStatuses: []v1.DeploymentStatus{
				{
					ID:           uuid.MustParse("77072d14-81bd-4e7a-b292-98be5ebefaf7"),
					DeploymentID: uuid.MustParse("f658aec8-0361-4f6c-ab10-1959ad433156"),
					CreatedAt:    now,
					State:        util.Ptr("created"),
				},
				{
					ID:           uuid.MustParse("f15929e6-7391-4bcd-9711-f78248390ed3"),
					DeploymentID: uuid.MustParse("f658aec8-0361-4f6c-ab10-1959ad433156"),

					CreatedAt: now.Add(time.Duration(time.Minute * 3)),
					State:     util.Ptr("waiting"),
				},
				{
					ID:           uuid.MustParse("5b5be8d6-30b4-47f1-9ae6-7bab79481ced"),
					DeploymentID: uuid.MustParse("f658aec8-0361-4f6c-ab10-1959ad433156"),
					CreatedAt:    now.Add(time.Duration(time.Minute * 5)),
					State:        util.Ptr("running"),
				},
			},
			expected: v1.Deployment{
				ID: uuid.MustParse("f658aec8-0361-4f6c-ab10-1959ad433156"),
				Status: v1.DeploymentStatus{
					ID:           uuid.MustParse("5b5be8d6-30b4-47f1-9ae6-7bab79481ced"),
					DeploymentID: uuid.MustParse("f658aec8-0361-4f6c-ab10-1959ad433156"),
					CreatedAt:    now.Add(time.Duration(time.Minute * 5)),
					State:        util.Ptr("running"),
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
			db.AutoMigrate(&v1.Deployment{})
			db.AutoMigrate(&v1.DeploymentStatus{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}
			for _, deploymentStatus := range tc.deploymentStatuses {
				err := db.Create(&deploymentStatus).Error
				if err != nil {
					t.Fatalf("unxepected error creating deployment status in test database: %s", err)
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
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			b, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.Deployment
			err = json.Unmarshal(b, &actual)

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("difference between actual and expected: %s", cmp.Diff(tc.expected, actual))
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
		deploymentStatuses []v1.DeploymentStatus
		clustermockOptions []clustermock.MockOption
		expected           []v1.Deployment
	}{
		{
			name:      "test single deployment",
			clusterID: "cluster-123",
			deployments: []v1.Deployment{
				{
					ID:        uuid.MustParse("115590c5-c5f5-48d3-95b4-5fd6a1d3e77f"),
					Name:      util.Ptr("test"),
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
					Name:      util.Ptr("test"),
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
					Name:      util.Ptr("test1"),
					ClusterID: "cluster-123",
				},
				{
					ID:        uuid.MustParse("d40c37d3-7465-4bc6-bfbf-19669f05a16a"),
					Name:      util.Ptr("test2"),
					ClusterID: "cluster-234",
				},
				{
					ID:        uuid.MustParse("a7743bee-d4cc-4342-b7bd-d149fa26f38f"),
					Name:      util.Ptr("test3"),
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
					Name:      util.Ptr("test1"),
					ClusterID: "cluster-123",
				},
				{
					ID:        uuid.MustParse("a7743bee-d4cc-4342-b7bd-d149fa26f38f"),
					Name:      util.Ptr("test3"),
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
					Name:      util.Ptr("test1"),
					ClusterID: "cluster-234",
				},
				{
					ID:        uuid.MustParse("1748bcf1-92c7-482e-a07c-a808701b2d84"),
					Name:      util.Ptr("test2"),
					ClusterID: "cluster-345",
				},
				{
					ID:        uuid.MustParse("fd9786ad-6722-4ac4-9e18-6a128472eb60"),
					Name:      util.Ptr("test3"),
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
		{
			name:      "test with deployment status",
			clusterID: "cluster-123",
			deployments: []v1.Deployment{
				{
					ID:        uuid.MustParse("2a0d2f6d-e3b1-4021-84cd-5c47918dc99e"),
					ClusterID: "cluster-123",
				},
			},
			deploymentStatuses: []v1.DeploymentStatus{
				{
					ID:           uuid.MustParse("dce9a76b-1a68-4d5d-bcea-fef85a265882"),
					DeploymentID: uuid.MustParse("fe9c90d4-6c0d-4038-8099-e4075bc1484b"),
					State:        util.Ptr("testing"),
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
					ID:        uuid.MustParse("2a0d2f6d-e3b1-4021-84cd-5c47918dc99e"),
					ClusterID: "cluster-123",
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
			db.AutoMigrate(&v1.Deployment{})
			db.AutoMigrate(&v1.DeploymentStatus{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}
			for _, deploymentStatus := range tc.deploymentStatuses {
				err := db.Create(&deploymentStatus).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment status in test database: %s", err)
				}
			}

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

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(actual, tc.expected))

			}
		})
	}
}

func TestDeleteDeployment(t *testing.T) {
	tt := []struct {
		name               string
		deploymentID       string
		deployments        []v1.Deployment
		deploymentStatuses []v1.DeploymentStatus
	}{
		{
			name:         "test single",
			deploymentID: "33de82a0-4133-45dc-b319-ab6a8a1daebc",
			deployments: []v1.Deployment{
				{
					ID:   uuid.MustParse("33de82a0-4133-45dc-b319-ab6a8a1daebc"),
					Name: util.Ptr("test-123"),
				},
			},
		},
		{
			name:         "test single with deployment statuses",
			deploymentID: "4be60902-c107-4485-8223-3179d666570d",
			deployments: []v1.Deployment{
				{
					ID: uuid.MustParse("4be60902-c107-4485-8223-3179d666570d"),
				},
			},
			deploymentStatuses: []v1.DeploymentStatus{
				{
					ID:           uuid.MustParse("de700fd4-386e-4384-8efb-47964102c51a"),
					DeploymentID: uuid.MustParse("4be60902-c107-4485-8223-3179d666570d"),
				},
				{
					ID:           uuid.MustParse("c6a72e21-46b1-46a3-a8eb-c11fc67e7152"),
					DeploymentID: uuid.MustParse("4be60902-c107-4485-8223-3179d666570d"),
				},
				{
					ID:           uuid.MustParse("1d646caf-3cec-4092-ac73-4badd0e31565"),
					DeploymentID: uuid.MustParse("4be60902-c107-4485-8223-3179d666570d"),
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Deployment{})
			db.AutoMigrate(&v1.DeploymentStatus{})

			for _, deployment := range tc.deployments {
				err := db.Create(&deployment).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment in test database: %s", err)
				}
			}
			for _, deploymentStatus := range tc.deploymentStatuses {
				err := db.Create(&deploymentStatus).Error
				if err != nil {
					t.Fatalf("unexpected error creating deployment status in database: %s", err)
				}
			}

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

			var deploymentStatuses []v1.DeploymentStatus
			err = db.Find(&deploymentStatuses, "deployment_id = ?", tc.deploymentID).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("unexpectede error finding deployment statuses in database: %s", err)
			}

			if len(deploymentStatuses) != 0 {
				t.Errorf("expected %d deployment statuses after delete, got %d", 0, len(deploymentStatuses))
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
				HelmChart: util.Ptr("test"),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Deployment{})
			db.AutoMigrate(&v1.DeploymentStatus{})

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
				Name: util.Ptr("InvalidName"),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test invalid container image",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				Name:           util.Ptr("test"),
				ContainerImage: util.Ptr("http://localhost:1234/my-image"),
			},
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:      "test name already in-use",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				Name: util.Ptr("test"),
			},
			existing: []v1.Deployment{
				{
					Name:      util.Ptr("test"),
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
				ContainerImage: util.Ptr("test"),
			},
		},
		{
			name:      "test port",
			clusterID: "cluster-123",
			deployment: v1.Deployment{
				ContainerImage: util.Ptr("nginx:l.2"),
				Port:           util.Ptr(1234),
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.Deployment{})
			db.AutoMigrate(&v1.DeploymentStatus{})

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
