package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
)

func TestValidateJSON(t *testing.T) {
	tt := []struct {
		name     string
		schema   string
		body     string
		expected int
	}{
		{
			name:     "test valid login",
			schema:   "#login",
			body:     `{"email":"test@dockyards.dev","password":"abc123"}`,
			expected: http.StatusOK,
		},
		{
			name:     "test login invalid field",
			schema:   "#login",
			body:     `{"email":"test@dockyards.dev","password":"abc123","test":true}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test valid cluster options",
			schema:   "#clusterOptions",
			body:     `{"name":"hello","cluster_template":"test"}`,
			expected: http.StatusOK,
		},
		{
			name:     "test cluster options invalid field",
			schema:   "#clusterOptions",
			body:     `{"name":"hello","invalid":true}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test cluster options nested invalid field",
			schema:   "#clusterOptions",
			body:     `{"name":"hello","node_pool_options":[{"disks":{}}]}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test valid workload",
			schema:   "#workload",
			body:     `{"namespace":"test","workload_template_name":"test"}`,
			expected: http.StatusOK,
		},
		{
			name:     "test workload missing namespace",
			schema:   "#workload",
			body:     `{"workload_template_name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload empty namespace",
			schema:   "#workload",
			body:     `{"namespace":"","workload_template_name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload invalid namespace prefix",
			schema:   "#workload",
			body:     `{"namespace":"-test","workload_template_name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload invalid namespace suffix",
			schema:   "#workload",
			body:     `{"namespace":"test-","workload_template_name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			validateJSON, err := middleware.NewValidateJSON("")
			if err != nil {
				t.Fatalf("error creating test middleware: %s", err)
			}

			schema := validateJSON.WithSchema(tc.schema)
			h := schema(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

			b := []byte(tc.body)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(b))

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			ctx := middleware.ContextWithLogger(r.Context(), logger)

			r = r.Clone(ctx)

			h.ServeHTTP(w, r)

			if w.Result().StatusCode != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, w.Result().StatusCode)
			}
		})
	}
}
