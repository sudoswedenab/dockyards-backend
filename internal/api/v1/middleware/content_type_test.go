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
