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
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
)

func TestContentType(t *testing.T) {
	t.Run("test application json", func(t *testing.T) {
		contentType := middleware.NewContentType("application/json").Handler

		h := contentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"test":true}`))
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		h.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}

		actual := w.Result().Header.Get("Content-Type")
		if actual != "application/json" {
			t.Errorf("expected %s, got %s", "application/json", actual)
		}
	})
}
