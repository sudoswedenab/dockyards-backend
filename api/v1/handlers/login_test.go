package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
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
		users       []v1.User
		login       v1.Login
		mockCluster types.ClusterService
		expected    int
	}{
		{
			name: "test valid user",
			users: []v1.User{
				{
					Email:    "test@dockyards.io",
					Password: string(hash),
				},
			},
			login: v1.Login{
				Email:    "test@dockyards.io",
				Password: "password",
			},
			expected: http.StatusOK,
		},
		{
			name: "test incorrect password",
			users: []v1.User{
				{
					Email:    "test@dockyards.io",
					Password: string(hash),
				},
			},
			login: v1.Login{
				Email:    "test@dockyards.io",
				Password: "incorrect",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "test missing user",
			login: v1.Login{
				Email:    "test@dockyards.io",
				Password: "password",
			},
			expected: http.StatusUnauthorized,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&v1.User{})

			for _, user := range tc.users {
				db.Create(&user)
			}

			h := handler{
				clusterService: tc.mockCluster,
				db:             db,
				logger:         logger,
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
