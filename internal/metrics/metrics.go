package metrics

import (
	"context"
	"log/slog"
	"runtime/debug"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/handlers"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type prometheusMetrics struct {
	logger             *slog.Logger
	registry           *prometheus.Registry
	organizationMetric *prometheus.GaugeVec
	userMetric         *prometheus.GaugeVec
	deploymentMetric   *prometheus.GaugeVec
	credentialMetric   *prometheus.GaugeVec
	controllerClient   client.Client
	clusterMetric      *prometheus.GaugeVec
}

type PrometheusMetricsOption func(*prometheusMetrics)

func WithLogger(logger *slog.Logger) PrometheusMetricsOption {
	return func(m *prometheusMetrics) {
		m.logger = logger
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

	clusterMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dockyards_backend_cluster",
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
		clusterMetric:      clusterMetric,
	}

	for _, prometheusMetricsOption := range prometheusMetricsOptions {
		prometheusMetricsOption(&m)
	}

	m.registry.MustRegister(m.organizationMetric)
	m.registry.MustRegister(m.userMetric)
	m.registry.MustRegister(m.deploymentMetric)
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

func (m *prometheusMetrics) CollectMetrics() error {
	ctx := context.Background()

	m.logger.Debug("collecting prometheus metrics")

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

	m.deploymentMetric.Reset()

	var deploymentList dockyardsv1.DeploymentList
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
		index.SecretTypeField: handlers.DockyardsSecretTypeCredential,
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

	var clusterList dockyardsv1.ClusterList
	err = m.controllerClient.List(ctx, &clusterList)
	if err != nil {
		m.logger.Error("error listing clusters", "err", err)
	}

	m.clusterMetric.Reset()

	for _, cluster := range clusterList.Items {
		ownerOrganizationName := getOwnerOrganizationName(&cluster)
		if ownerOrganizationName == "" {
			m.logger.Warn("cluster has no owner organization", "name", cluster.Name)

			continue
		}

		labels := prometheus.Labels{
			"name":              cluster.Name,
			"organization_name": ownerOrganizationName,
		}

		m.clusterMetric.With(labels).Set(1)
	}

	return nil
}

func getOwnerClusterUID(object client.Object) string {
	ownerReferences := object.GetOwnerReferences()
	for _, ownerReference := range ownerReferences {
		if ownerReference.APIVersion != dockyardsv1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != dockyardsv1.ClusterKind {
			continue
		}

		return string(ownerReference.UID)
	}

	return ""
}

func getOwnerOrganizationName(object client.Object) string {
	ownerReferences := object.GetOwnerReferences()
	for _, ownerReference := range ownerReferences {
		if ownerReference.APIVersion != dockyardsv1.GroupVersion.String() {
			continue
		}

		if ownerReference.Kind != dockyardsv1.OrganizationKind {
			continue
		}

		return ownerReference.Name
	}

	return ""
}
