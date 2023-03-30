package rancher

import (
	"os"
	"testing"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"golang.org/x/exp/slog"
)

func TestRancherCreateUser(t *testing.T) {
	tt := []struct {
		name     string
		user     model.RancherUser
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
			expected: "abc123",
		},
	}

	cattleUrl := os.Getenv("INTEGRATION_TEST_CATTLE_URL")
	bearerToken := os.Getenv("INTEGRATION_TEST_BEARER_TOKEN")

	if cattleUrl == "" {
		t.Skip("Internal test cattle url not set")
	}

	if bearerToken == "" {
		t.Skip("Internal test bearer token not set")
	}

	logger := slog.New(slog.HandlerOptions{Level: slog.LevelError + 1}.NewTextHandler(os.Stdout))

	r, err := NewRancher(bearerToken, cattleUrl, logger, true)
	if err != nil {
		t.Fatalf("unexpected error creating new rancher: %s", err)
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
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
		expected string
	}{
		{
			name:     "test empty",
			expected: "unexpected status code 400, data: empty request",
		},
		{
			name: "test short password",
			user: model.RancherUser{
				Password: "hello",
			},
			expected: "unexpected status code 400, data: password too short",
		},
		{
			name:     "test missing dockyard-role",
			expected: "no role named 'dockyard-role' found",
		},
		{
			name:     "test role binding internal server error",
			expected: "unexpected status code 500, data: test",
		},
	}

	cattleUrl := os.Getenv("INTEGRATION_TEST_CATTLE_URL")
	bearerToken := os.Getenv("INTEGRATION_TEST_BEARER_TOKEN")

	if cattleUrl == "" {
		t.Skip("Internal test cattle url not set")
	}

	if bearerToken == "" {
		t.Skip("Internal test bearer token not set")
	}

	logger := slog.New(slog.HandlerOptions{Level: slog.LevelError + 1}.NewTextHandler(os.Stdout))

	r, err := NewRancher(bearerToken, cattleUrl, logger, true)
	if err != nil {
		t.Fatalf("unexpected error creating new rancher: %s", err)
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
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
