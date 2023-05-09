package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSignup(t *testing.T) {
	tt := []struct {
		name        string
		mockCluster types.ClusterService
		signup      model.Signup
		expected    int
	}{
		{
			name: "test success",
			signup: model.Signup{
				Name:     "test",
				Email:    "test@dockyards.io",
				Password: "hello",
			},
			expected: http.StatusCreated,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&model.User{})

			h := handler{
				clusterService: tc.mockCluster,
				db:             db,
			}

			r := gin.New()
			r.POST("/test", h.Signup)

			b, err := json.Marshal(tc.signup)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}

			r.ServeHTTP(w, req)

			if w.Code != tc.expected {
				t.Errorf("expected code %d, got %d", tc.expected, w.Code)
			}
		})
	}
}
