package handlers

import (
	"log/slog"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRegisterRoutes(t *testing.T) {
	tt := []struct {
		name     string
		expected error
	}{
		{
			name: "test empty",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			tokens := WithJWTAccessTokens([]byte("access123"), []byte("refresh123"))

			actual := RegisterRoutes(r, db, logger, tokens)
			if actual != tc.expected {
				t.Errorf("expected error %s, got %s", tc.expected, actual)
			}
		})
	}
}
