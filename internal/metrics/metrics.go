package metrics

import (
	"log/slog"
	"runtime/debug"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type prometheusMetrics struct {
	logger             *slog.Logger
	db                 *gorm.DB
	registry           *prometheus.Registry
	organizationMetric *prometheus.GaugeVec
	userMetric         *prometheus.GaugeVec
	deploymentMetric   *prometheus.GaugeVec
	credentialMetric   *prometheus.GaugeVec
}

type PrometheusMetricsOption func(*prometheusMetrics)

func WithLogger(logger *slog.Logger) PrometheusMetricsOption {
	return func(m *prometheusMetrics) {
		m.logger = logger
	}
}

func WithDatabase(db *gorm.DB) PrometheusMetricsOption {
	return func(m *prometheusMetrics) {
		m.db = db
	}
}

func WithPrometheusRegistry(registry *prometheus.Registry) PrometheusMetricsOption {
	return func(m *prometheusMetrics) {
		m.registry = registry
	}
}

func NewPrometheusMetrics(prometheusMetricsOptions ...PrometheusMetricsOption) (*prometheusMetrics, error) {

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

	deploymentMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_deployment",
		},
		[]string{
			"name",
			"cluster_id",
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

	m := prometheusMetrics{
		organizationMetric: organizationMetric,
		userMetric:         userMetric,
		deploymentMetric:   deploymentMetric,
		credentialMetric:   credentialMetric,
	}

	for _, prometheusMetricsOption := range prometheusMetricsOptions {
		prometheusMetricsOption(&m)
	}

	m.registry.MustRegister(m.organizationMetric)
	m.registry.MustRegister(m.userMetric)
	m.registry.MustRegister(m.deploymentMetric)
	m.registry.MustRegister(m.credentialMetric)

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

func (m *prometheusMetrics) CollectMetrics() error {
	m.logger.Debug("collecting prometheus metrics from database")

	var organizations []v1.Organization
	err := m.db.Find(&organizations).Error
	if err != nil {
		m.logger.Error("error finding organizations in database", "err", err)
		return err
	}

	m.organizationMetric.Reset()

	for _, organization := range organizations {
		labels := prometheus.Labels{
			"name": organization.Name,
		}

		m.organizationMetric.With(labels).Set(1)
	}

	m.userMetric.Reset()

	var users []v1.User
	err = m.db.Find(&users).Error
	if err != nil {
		m.logger.Error("error finding users in database", "err", err)
		return err
	}

	for _, user := range users {
		labels := prometheus.Labels{
			"name": user.Name,
		}

		m.userMetric.With(labels).Set(1)
	}

	m.deploymentMetric.Reset()

	var deployments []v1.Deployment
	err = m.db.Find(&deployments).Error
	if err != nil {
		m.logger.Error("error finding deployments in database", "err", err)
		return err
	}

	for _, deployment := range deployments {
		labels := prometheus.Labels{
			"name":       *deployment.Name,
			"cluster_id": deployment.ClusterID,
		}

		m.deploymentMetric.With(labels).Set(1)
	}

	m.credentialMetric.Reset()

	var credentials []v1.Credential
	err = m.db.Find(&credentials).Error
	if err != nil {
		m.logger.Error("error finding credentials in database", "err", err)
		return err
	}

	for _, credential := range credentials {
		labels := prometheus.Labels{
			"name":              credential.Name,
			"organization_name": credential.Organization,
		}

		m.credentialMetric.With(labels).Set(1)
	}

	return nil
}
