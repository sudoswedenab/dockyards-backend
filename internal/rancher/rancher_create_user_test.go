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
				t.Logf("r: %#v", r)
				switch r.URL.Path {
				case "/v3/users":
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(`{"id":"abc123"}`))
				case "/v3/globalRoles":
					w.Write([]byte(`{"data":[{"name":"dockyard-role","id":"role123"}]}`))
				case "/v3/globalrolebindings":
					w.WriteHeader(http.StatusCreated)
				case "/":
					w.Header().Add("X-API-Schemas", r.Host)
				}
			},
			expected: "abc123",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.handler))

			r, err := NewRancher("bearer-token", ts.URL)
			if err != nil {
				t.Fatalf("unexpected error creating new rancher: %s", err)
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
		name     string
		user     model.RancherUser
		handler  func(http.ResponseWriter, *http.Request)
		expected string
	}{
		{
			name: "test empty",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("empty request"))
			},
			expected: "unexpected status code 400, data: empty request",
		},
		{
			name: "test short password",
			user: model.RancherUser{
				Password: "hello",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("password too short"))
			},
			expected: "unexpected status code 400, data: password too short",
		},
		{
			name: "test missing dockyard-role",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v3/users":
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(`{"id":"user123"}`))
				case "/v3/globalRoles":
					w.Write([]byte(`{"data":[]}`))
				default:
					w.WriteHeader(http.StatusTeapot)
				}
			},
			expected: "no role named 'dockyard-role' found",
		},
		{
			name: "test role binding internal server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v3/users":
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(`{"id":"user123"}`))
				case "/v3/globalRoles":
					w.Write([]byte(`{"data":[{"name":"dockyard-role","id":"role123"}]}`))
				case "/v3/globalrolebindings":
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("test"))
				default:
					w.WriteHeader(http.StatusTeapot)
				}
			},
			expected: "unexpected status code 500, data: test",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tc.handler))

			r, err := NewRancher("bearer-token", ts.URL)
			if err != nil {
				t.Fatalf("unexpected error creating new rancher: %s", err)
			}

			_, err = r.RancherCreateUser(tc.user)
			if err == nil {
				t.Fatalf("expected to get error, got nil")
			}

			if err.Error() != tc.expected {
				t.Errorf("expected to get string '%s', got '%s'", tc.expected, err.Error())
			}
		})
	}
}
