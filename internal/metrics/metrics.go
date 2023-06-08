package metrics

import (
	"runtime/debug"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type prometheusMetrics struct {
	logger             *slog.Logger
	db                 *gorm.DB
	registry           *prometheus.Registry
	organizationMetric *prometheus.GaugeVec
	userMetric         *prometheus.GaugeVec
	appMetric          *prometheus.GaugeVec
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

	appMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_app",
		},
		[]string{
			"name",
			"organization_name",
			"cluster_name",
		},
	)

	m := prometheusMetrics{
		organizationMetric: organizationMetric,
		userMetric:         userMetric,
		appMetric:          appMetric,
	}

	for _, prometheusMetricsOption := range prometheusMetricsOptions {
		prometheusMetricsOption(&m)
	}

	m.registry.MustRegister(m.organizationMetric)
	m.registry.MustRegister(m.userMetric)
	m.registry.MustRegister(m.appMetric)

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

	var organizations []model.Organization
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

	var users []model.User
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

	m.appMetric.Reset()

	var apps []model.App
	err = m.db.Find(&apps).Error
	if err != nil {
		m.logger.Error("error finding apps in database", "err", err)
		return err
	}

	for _, app := range apps {
		labels := prometheus.Labels{
			"name":              app.Name,
			"organization_name": app.Organization,
			"cluster_name":      app.Cluster,
		}

		m.appMetric.With(labels).Set(1)
	}

	return nil
}
