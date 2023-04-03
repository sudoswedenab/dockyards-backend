package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestLogin(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
	if err != nil {
		t.Fatalf("unexpected error hashing string 'password'")
	}

	tt := []struct {
		name        string
		users       []model.User
		login       model.Login
		mockCluster types.ClusterService
		expected    int
	}{
		{
			name: "test valid user",
			users: []model.User{
				{
					Email:    "test@dockyards.io",
					Password: string(hash),
				},
			},
			login: model.Login{
				Email:    "test@dockyards.io",
				Password: "password",
			},
			expected: http.StatusOK,
		},
		{
			name: "test incorrect password",
			users: []model.User{
				{
					Email:    "test@dockyards.io",
					Password: string(hash),
				},
			},
			login: model.Login{
				Email:    "test@dockyards.io",
				Password: "incorrect",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "test missing user",
			login: model.Login{
				Email:    "test@dockyards.io",
				Password: "password",
			},
			expected: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&model.User{})

			for _, user := range tc.users {
				db.Create(&user)
			}

			h := handler{
				clusterService: tc.mockCluster,
				db:             db,
			}

			r := gin.New()
			r.POST("/test", h.Login)

			b, err := json.Marshal(tc.login)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}
			req.Header.Add("content-type", "application/json")

			r.ServeHTTP(w, req)

			body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			if w.Code != tc.expected {
				t.Errorf("expected code %d, got %d", tc.expected, w.Code)
				t.Log("body:", string(body))
			}
		})
	}
}
