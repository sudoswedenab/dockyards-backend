package handlers

import (
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
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/go-cmp/cmp"
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
