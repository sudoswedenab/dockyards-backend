package handlers

import (
	"log/slog"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
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

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			actual := RegisterRoutes(r, logger)
			if actual != tc.expected {
				t.Errorf("expected error %s, got %s", tc.expected, actual)
			}
		})
	}
}
