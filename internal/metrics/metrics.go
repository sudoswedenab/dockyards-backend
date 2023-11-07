package metrics

import (
	"context"
	"log/slog"
	"runtime/debug"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/handlers"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type prometheusMetrics struct {
	logger             *slog.Logger
	db                 *gorm.DB
	registry           *prometheus.Registry
	organizationMetric *prometheus.GaugeVec
	userMetric         *prometheus.GaugeVec
	deploymentMetric   *prometheus.GaugeVec
	credentialMetric   *prometheus.GaugeVec
	controllerClient   client.Client
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

func WithManager(manager ctrl.Manager) PrometheusMetricsOption {
	controllerClient := manager.GetClient()

	return func(m *prometheusMetrics) {
		m.controllerClient = controllerClient
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
	ctx := context.Background()

	m.logger.Debug("collecting prometheus metrics")

	var organizationList v1alpha1.OrganizationList
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

	var userList v1alpha1.UserList
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

	m.deploymentMetric.Reset()

	var deploymentList v1alpha1.DeploymentList
	err = m.controllerClient.List(ctx, &deploymentList)
	if err != nil {
		m.logger.Error("error listing deployments", "err", err)

		return err
	}

	for _, deployment := range deploymentList.Items {
		ownerClusterUID := getOwnerClusterUID(&deployment)
		if ownerClusterUID == "" {
			m.logger.Warn("deployment has no owner cluster", "name", deployment.Name)

			continue
		}

		labels := prometheus.Labels{
			"name":       deployment.Name,
			"cluster_id": ownerClusterUID,
		}

		m.deploymentMetric.With(labels).Set(1)
	}

	m.credentialMetric.Reset()

	matchingFields := client.MatchingFields{
		index.SecretTypeIndexKey: handlers.DockyardsSecretTypeCredential,
	}

	var secretList corev1.SecretList
	err = m.controllerClient.List(ctx, &secretList, matchingFields)
	if err != nil {
		m.logger.Error("error listing secrets", "err", err)

		return err
	}

	for _, secret := range secretList.Items {
		ownerOrganizationName := getOwnerOrganizationName(&secret)
		if ownerOrganizationName == "" {
			m.logger.Warn("secret has no owner organization", "name", secret.Name)

			continue
		}

		labels := prometheus.Labels{
			"name":              secret.Name,
			"organization_name": ownerOrganizationName,
		}

		m.credentialMetric.With(labels).Set(1)
	}

	return nil
}

func getOwnerClusterUID(object client.Object) string {
	ownerReferences := object.GetOwnerReferences()
	for _, ownerReference := range ownerReferences {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != v1alpha1.ClusterKind {
			continue
		}

		return string(ownerReference.UID)
	}

	return ""
}

func getOwnerOrganizationName(object client.Object) string {
	ownerReferences := object.GetOwnerReferences()
	for _, ownerReference := range ownerReferences {
		if ownerReference.APIVersion != v1alpha1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != v1alpha1.OrganizationKind {
			continue
		}

		return ownerReference.Name
	}

	return ""
}
