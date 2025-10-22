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

package config_test

import (
	"context"
	"testing"

	"github.com/sudoswedenab/dockyards-backend/api/config"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()

	scheme := scheme.Scheme
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	cmName := "dockyards-config"
	cmNamespace := "dockyards-system"
	url := "http://test.com"

	configMap := corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      cmName,
			Namespace: cmNamespace,
		},
		Data: map[string]string{
			"externalUrl": url,
		},
	}

	err := fakeClient.Create(ctx, &configMap)
	if err != nil {
		t.Fatalf("error creating config map: %s", err)
	}

	t.Run("get dockyards config", func(t *testing.T) {
		_, err := config.GetConfig(ctx, fakeClient, cmName, cmNamespace)
		if err != nil {
			t.Fatalf("error getting config: %s", err)
		}
	})

	t.Run("get key from dockyards config", func(t *testing.T) {
		c, err := config.GetConfig(ctx, fakeClient, cmName, cmNamespace)
		if err != nil {
			t.Fatalf("error getting config: %s", err)
		}

		eUrl := c.GetConfigKey("externalUrl", "http://default.com")
		if eUrl != url {
			t.Fatalf("The externalUrl does not match the expected value: expected \"%s\", got \"%s\"", url, eUrl)
		}
	})

	t.Run("get default for key from dockyards config", func(t *testing.T) {
		c, err := config.GetConfig(ctx, fakeClient, cmName, cmNamespace)
		if err != nil {
			t.Fatalf("error getting config: %s", err)
		}

		defaultValue := "defaultForNonExistent"

		nonExistent := c.GetConfigKey("nonExistent", defaultValue)
		if nonExistent != defaultValue {
			t.Fatalf("The non-existent key does not match the default value: expected \"%s\", got \"%s\"", defaultValue, nonExistent)
		}
	})

	t.Run("set a key in dockyards config", func(t *testing.T) {
		c, err := config.GetConfig(ctx, fakeClient, cmName, cmNamespace)
		if err != nil {
			t.Fatalf("error getting config: %s", err)
		}

		oldValue := c.GetConfigKey("externalUrl", "http://default.com")

		err = c.SetConfigKey(ctx, fakeClient, "externalUrl", "http://new.com")
		if err != nil {
			t.Fatalf("error setting key in config: %s", err)
		}

		newValue := c.GetConfigKey("externalUrl", "http://default.com")
		if oldValue == newValue {
			t.Fatalf("The key was not updated in kubernetes ConfigMap")
		}
	})
}
