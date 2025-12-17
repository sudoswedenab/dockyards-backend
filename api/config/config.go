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

package config

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DockyardsConfigReader interface {
	GetConfigKey(key string, defaultValue string) string
}

type DockyardsConfigWriter interface {
	SetConfigKey(ctx context.Context, client client.Client, key string, value string) error
}

// DockyardsConfig holds the configuration data for a Dockyards installation,
// including its name, namespace, and a map of configuration settings.
type dockyardsConfig struct {
	name      string
	namespace string
	config    map[string]string
	mutex     sync.Mutex
}

// GetConfig retrieves the configuration from the specified ConfigMap in the given namespace
// and returns it as a DockyardsConfig instance.
func GetConfig(ctx context.Context, c client.Client, configMap, dockyardsNamespace string) (*dockyardsConfig, error) {
	config := dockyardsConfig{
		name:      configMap,
		namespace: dockyardsNamespace,
	}

	cm := corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Name: configMap, Namespace: dockyardsNamespace}, &cm)
	if err != nil {
		return &dockyardsConfig{}, err
	}

	config.config = cm.Data

	return &config, nil
}

// GetConfigKey retrieves the value for the specified key from the configuration,
// returning the provided defaultValue if the key is not present or the config is nil.
func (config *dockyardsConfig) GetConfigKey(key, defaultValue string) string {
	c := (*config).config
	if c == nil || c[key] == "" {
		return defaultValue
	}

	return c[key]
}

// SetConfigKey sets the value for the specified key in the configuration and updates the ConfigMap
// in the Kubernetes cluster.
func (config *dockyardsConfig) SetConfigKey(ctx context.Context, c client.Client, key, value string) error {
	config.mutex.Lock()
	defer config.mutex.Unlock()

	if config.config == nil {
		config.config = make(map[string]string)
	}

	config.config[key] = value
	err := config.setConfig(ctx, c)
	if err != nil {
		return err
	}

	return nil
}

// setConfig updates the ConfigMap in the Kubernetes cluster with the current configuration data.
func (config *dockyardsConfig) setConfig(ctx context.Context, c client.Client) error {
	cm := corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      config.name,
			Namespace: config.namespace,
		},
	}
	err := c.Get(ctx, client.ObjectKeyFromObject(&cm), &cm)
	if err != nil {
		return err
	}

	cm.Data = config.config

	err = c.Update(ctx, &cm)
	if err != nil {
		return err
	}

	return nil
}

type DefaultingConfig struct {}

func (c *DefaultingConfig) GetConfigKey(_, defaultValue string) string {
	return defaultValue
}
