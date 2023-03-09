package rancher

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/sudosweden/backend/api/v1/model"
)

func TestRancherCreateUser(t *testing.T) {
	tt := []struct {
		name     string
		user     model.RancherUser
		handler  func(http.ResponseWriter, *http.Request)
		expected string
	}{
		{
			name: "test basic",
			user: model.RancherUser{
				Username:           "basic",
				Name:               "basic",
				Password:           "testbasic123",
				MustChangePassword: false,
				Enabled:            true,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("{\"id\":\"abc123\"}"))
			},
			expected: "abc123",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.handler))

			r := Rancher{
				Url: ts.URL,
			}

			actual, err := r.RancherCreateUser(tc.user)
			if err != nil {
				t.Errorf("expected error to be nil, got %s", err)
			}

			if actual != tc.expected {
				t.Errorf("expected to get id %s, got %s", tc.expected, actual)
			}
		})
	}
}

func TestRancherCreateUserErrors(t *testing.T) {
	tt := []struct {
		name    string
		user    model.RancherUser
		handler func(http.ResponseWriter, *http.Request)
	}{
		{
			name: "test empty",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
		},
		{
			name: "test short password",
			user: model.RancherUser{
				Password: "hello",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.handler))

			r := Rancher{
				Url: ts.URL,
			}
			_, err := r.RancherCreateUser(tc.user)
			if err == nil {
				t.Errorf("expected to get error, got nil")
			}
		})
	}
}
