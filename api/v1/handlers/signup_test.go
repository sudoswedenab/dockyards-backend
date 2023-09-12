package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestSignup(t *testing.T) {
	tt := []struct {
		name     string
		signup   v1.Signup
		expected int
	}{
		{
			name: "test success",
			signup: v1.Signup{
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
			db.AutoMigrate(&v1.User{})

			h := handler{
				db: db,
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

func TestSignupErrors(t *testing.T) {
	tt := []struct {
		name     string
		signup   v1.Signup
		users    []v1.User
		expected int
	}{
		{
			name:     "test empty",
			expected: http.StatusUnprocessableEntity,
		},
		{
			name: "test existing name",
			signup: v1.Signup{
				Name: "test",
			},
			users: []v1.User{
				{
					ID:   uuid.MustParse("bda0a6d4-4ef5-42fc-90d9-25ae95b17626"),
					Name: "test",
				},
			},
			expected: http.StatusConflict,
		},
		{
			name: "test existing email",
			signup: v1.Signup{
				Name:  "test",
				Email: "test@dockyards.dev",
			},
			users: []v1.User{
				{
					ID:    uuid.MustParse("fc8b53cb-0880-4232-99e0-a9dadf0e7b74"),
					Name:  "user",
					Email: "test@dockyards.dev",
				},
			},
			expected: http.StatusConflict,
		},
		{
			name: "test long password",
			signup: v1.Signup{
				Name:     "test",
				Email:    "test@dockyards.dev",
				Password: "45744813965d5cd840e2c1ba2a16c82b4d0f2eff1872de0b956e191180179216e19e162ff76d00d112a79527ebdefaa7273a44107c8868047fabd6f7e9080cf0fd65773e1dc2999275",
			},
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger, TranslateError: true})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&v1.User{})
			for _, user := range tc.users {
				err := db.Create(&user).Error
				if err != nil {
					t.Fatalf("unexpected error creating user in test database: %s", err)
				}
			}

			h := handler{
				logger: logger,
				db:     db,
			}

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			r.POST("/v1/signup", h.Signup)

			b, err := json.Marshal(tc.signup)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			c.Request, err = http.NewRequest(http.MethodPost, "/v1/signup", bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unxepected error creating new request: %s", err)
			}

			r.ServeHTTP(w, c.Request)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}
