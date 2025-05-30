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

package metrics

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PrometheusMetrics struct {
	logger             *slog.Logger
	registry           *prometheus.Registry
	organizationMetric *prometheus.GaugeVec
	userMetric         *prometheus.GaugeVec
	credentialMetric   *prometheus.GaugeVec
	controllerClient   client.Client
	clusterMetric      *prometheus.GaugeVec
}

type PrometheusMetricsOption func(*PrometheusMetrics)

func WithLogger(logger *slog.Logger) PrometheusMetricsOption {
	return func(m *PrometheusMetrics) {
		m.logger = logger
	}
}

func WithPrometheusRegistry(registry *prometheus.Registry) PrometheusMetricsOption {
	return func(m *PrometheusMetrics) {
		m.registry = registry
	}
}

func WithManager(mgr ctrl.Manager) PrometheusMetricsOption {
	controllerClient := mgr.GetClient()

	return func(m *PrometheusMetrics) {
		m.controllerClient = controllerClient
	}
}

func NewPrometheusMetrics(prometheusMetricsOptions ...PrometheusMetricsOption) (*PrometheusMetrics, error) {
	organizationMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_organization",
		},
		[]string{
			"name",
		},
	)

	userMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_user",
		},
		[]string{
			"name",
		},
	)

	credentialMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_credential",
		},
		[]string{
			"name",
			"organization_name",
		},
	)

	clusterMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_cluster",
		},
		[]string{
			"name",
			"organization_name",
		},
	)

	m := PrometheusMetrics{
		organizationMetric: organizationMetric,
		userMetric:         userMetric,
		credentialMetric:   credentialMetric,
		clusterMetric:      clusterMetric,
	}

	for _, PrometheusMetricsOption := range prometheusMetricsOptions {
		PrometheusMetricsOption(&m)
	}

	m.registry.MustRegister(m.organizationMetric)
	m.registry.MustRegister(m.userMetric)
	m.registry.MustRegister(m.credentialMetric)
	m.registry.MustRegister(m.clusterMetric)

	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		buildMetric := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dockyards_backend_build_info",
			},
			[]string{
				"goversion",
				"revision",
			},
		)

		revision := "(unknown)"
		for _, setting := range buildInfo.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
		}

		labels := prometheus.Labels{
			"goversion": buildInfo.GoVersion,
			"revision":  revision,
		}

		buildMetric.With(labels).Inc()

		m.registry.MustRegister(buildMetric)
	}

	return &m, nil
}

func (m *PrometheusMetrics) CollectMetrics() error {
	ctx := context.Background()

	m.logger.Log(ctx, slog.LevelDebug-1, "collecting prometheus metrics")

	var organizationList dockyardsv1.OrganizationList
	err := m.controllerClient.List(ctx, &organizationList)
	if err != nil {
		m.logger.Error("error listing organizations in kubernetes", "err", err)

		return err
	}

	m.organizationMetric.Reset()

	for _, organization := range organizationList.Items {
		labels := prometheus.Labels{
			"name": organization.Name,
		}

		m.organizationMetric.With(labels).Set(1)
	}

	m.userMetric.Reset()

	var userList dockyardsv1.UserList
	err = m.controllerClient.List(ctx, &userList)
	if err != nil {
		m.logger.Error("error finding users in kubernetes", "err", err)

		return err
	}

	for _, user := range userList.Items {
		labels := prometheus.Labels{
			"name": user.Name,
		}

		m.userMetric.With(labels).Set(1)
	}

	m.credentialMetric.Reset()

	matchingFields := client.MatchingFields{
		index.SecretTypeField: dockyardsv1.SecretTypeCredential,
	}

	var secretList corev1.SecretList
	err = m.controllerClient.List(ctx, &secretList, matchingFields)
	if err != nil {
		m.logger.Error("error listing secrets", "err", err)

		return err
	}

	for _, secret := range secretList.Items {
		ownerOrganization, err := apiutil.GetOwnerOrganization(ctx, m.controllerClient, &secret)
		if err != nil {
			m.logger.Warn("error getting owner organization", "err", err)

			continue
		}

		if ownerOrganization == nil {
			m.logger.Warn("secret has no owner organization", "name", secret.Name, "namespace", secret.Namespace)

			continue
		}

		labels := prometheus.Labels{
			"name":              secret.Name,
			"organization_name": ownerOrganization.Name,
		}

		m.credentialMetric.With(labels).Set(1)
	}

	var clusterList dockyardsv1.ClusterList
	err = m.controllerClient.List(ctx, &clusterList)
	if err != nil {
		m.logger.Error("error listing clusters", "err", err)
	}

	m.clusterMetric.Reset()

	for _, cluster := range clusterList.Items {
		ownerOrganization, err := apiutil.GetOwnerOrganization(ctx, m.controllerClient, &cluster)
		if err != nil {
			m.logger.Warn("error getting owner organization", "err", err)

			continue
		}

		if ownerOrganization == nil {
			m.logger.Warn("cluster has no owner organization", "name", cluster.Name)

			continue
		}

		labels := prometheus.Labels{
			"name":              cluster.Name,
			"organization_name": ownerOrganization.Name,
		}

		m.clusterMetric.With(labels).Set(1)
	}

	return nil
}
