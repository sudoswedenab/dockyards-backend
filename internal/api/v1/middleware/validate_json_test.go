// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
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
			schema:   "#workloadOptions",
			body:     `{"namespace":"test","workload_template_name":"test","name":"test"}`,
			expected: http.StatusOK,
		},
		{
			name:     "test workload missing name",
			schema:   "#workloadOptions",
			body:     `{"workload_template_name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload empty namespace",
			schema:   "#workloadOptions",
			body:     `{"namespace":"","workload_template_name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload invalid namespace prefix",
			schema:   "#workloadOptions",
			body:     `{"namespace":"-test","workload_template_name":"test","name":test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload invalid namespace suffix",
			schema:   "#workloadOptions",
			body:     `{"namespace":"test-","workload_template_name":"test","name":"test"}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test workload numeric suffix",
			schema:   "#workloadOptions",
			body:     `{"namespace":"0-test","workload_template_name":"test","name":"test"}`,
			expected: http.StatusOK,
		},
		{
			name:     "test workload numeric suffix",
			schema:   "#workloadOptions",
			body:     `{"namespace":"test-0","workload_template_name":"test","name":"test"}`,
			expected: http.StatusOK,
		},
		{
			name:     "test node pool options valid quantity",
			schema:   "#nodePoolOptions",
			body:     `{"name":"test","quantity":3}`,
			expected: http.StatusOK,
		},
		{
			name:     "test node pool options negative quantity",
			schema:   "#nodePoolOptions",
			body:     `{"name":"test","quantity":-3}`,
			expected: http.StatusUnprocessableEntity,
		},
		{
			name:     "test node pool options storage resource",
			schema:   "#nodePoolOptions",
			body:     `{"name":"test","quantity":1,"storage_resources":[{"name":"test"}]}`,
			expected: http.StatusOK,
		},
		{
			name:     "test node pool options storage resource name",
			schema:   "#nodePoolOptions",
			body:     `{"name":"test","quantity":1,"storage_resources":[{"name":"Test"}]}`,
			expected: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			validateJSON, err := middleware.NewValidateJSON()
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
