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

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetConfig(ctx context.Context, c client.Client, configMap, dockyardsNamespace string) (map[string]string, error) {
	cm := corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Name: configMap, Namespace: dockyardsNamespace}, &cm)
	if err != nil {
		return nil, err
	}

	return cm.Data, nil
}

func GetConfigKey(config *map[string]string, key, defaultValue string) string {
	c := *config
	if c == nil || c[key] == "" {
		return defaultValue
	}

	return c[key]
}
