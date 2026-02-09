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
	"log/slog"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ConfigManager holds the configuration data of a config map.
// Updating the map externally is fine, ConfigManager will keep the
// its internal state in sync. You should not store the config values in
// data structures, but rather do lookups via GetValueForKey(), this lookup
// is very cheap.
type ConfigManager struct {
	client    client.Client
	backingConfigMapKey client.ObjectKey
	data      atomic.Pointer[map[string]string]
	logger    *slog.Logger
}

var _ reconcile.Reconciler = &ConfigManager{}

type ConfigManagerOption func(*ConfigManager)

func WithLogger(logger *slog.Logger) ConfigManagerOption {
	return func (m *ConfigManager) {
		m.logger = logger
	}
}

func NewConfigManager(mgr ctrl.Manager, backingConfigMapKey client.ObjectKey, options ...ConfigManagerOption) (*ConfigManager, error) {
	result := &ConfigManager{
		client: mgr.GetClient(),
		backingConfigMapKey: backingConfigMapKey,
	}

	for _, option := range options {
		option(result)
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func NewFakeConfigManager(data map[string]string) *ConfigManager {
	result := &ConfigManager{}
	result.data.Store(&data)
	return result
}

func (m *ConfigManager) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if req.NamespacedName != m.backingConfigMapKey {
		return reconcile.Result{}, nil
	}

	configMap := corev1.ConfigMap{}
	err := m.client.Get(ctx, m.backingConfigMapKey, &configMap)
	if err != nil {
		return reconcile.Result{}, err
	}
	m.data.Store(&configMap.Data)

	if m.logger != nil {
		m.logger.Debug("reloaded config map", "key", m.backingConfigMapKey, "data", configMap.Data)
	}

	return reconcile.Result{}, nil
}

func (m *ConfigManager) GetValueForKey(key Key) (string, bool) {
	configPtr := m.data.Load()
	if configPtr == nil {
		return "", false 
	}
	config := *configPtr
	if config == nil {
		return "", false 
	}
	value, ok := config[string(key)]
	return value, ok
}

func (m *ConfigManager) GetValueOrDefault(key Key, defaultValue string) string {
	value, ok := m.GetValueForKey(key)
	if !ok {
		return defaultValue
	}
	return value
}
