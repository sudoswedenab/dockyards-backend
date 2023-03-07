package rancher

import (
	"os"
	"testing"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
)

func TestRancherCreateUser(t *testing.T) {
	tt := []struct {
		name string
		user model.RancherUser
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
		},
	}

	internal.CattleBearerToken = os.Getenv("TEST_CATTLE_BEARER_TOKEN")
	internal.CattleUrl = "https://localhost:8443"

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RancherCreateUser(tc.user)
			if err != nil {
				t.Fatalf("unxepected error creating rancher user: %s", err)
			}
		})
	}
}

func TestRancherCreateUserErrors(t *testing.T) {
	tt := []struct {
		name string
		user model.RancherUser
	}{
		{
			name: "test empty",
		},
		{
			name: "test short password",
			user: model.RancherUser{
				Password: "hello",
			},
		},
	}

	internal.CattleBearerToken = os.Getenv("TEST_CATTLE_BEARER_TOKEN")
	internal.CattleUrl = "https://localhost:8443"

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RancherCreateUser(tc.user)
			if err == nil {
				t.Errorf("expected to get error, got nil")
			}

			t.Log("err:", err)
		})
	}
}
