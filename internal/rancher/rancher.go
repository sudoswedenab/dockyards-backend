package rancher

import (
	"sync"

	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rancher/norman/clientbase"
	normanTypes "github.com/rancher/norman/types"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type rancher struct {
	managementClient   *managementv3.Client
	clientOpts         *clientbase.ClientOpts
	logger             *slog.Logger
	garbageMutex       *sync.Mutex
	garbageObjects     map[string]*normanTypes.Resource
	cloudService       types.CloudService
	prometheusRegistry *prometheus.Registry
	clusterMetric      *prometheus.GaugeVec
	controllerClient   client.Client
}

var _ types.ClusterService = &rancher{}

type RancherOption func(*rancher)

func WithLogger(logger *slog.Logger) RancherOption {
	return func(r *rancher) {
		r.logger = logger
	}
}

func WithRancherClientOpts(url, tokenKey string, insecure bool) RancherOption {
	return func(r *rancher) {
		r.clientOpts = &clientbase.ClientOpts{
			URL:      url,
			TokenKey: tokenKey,
			Insecure: insecure,
		}
	}
}

func WithCloudService(cloudService types.CloudService) RancherOption {
	return func(r *rancher) {
		r.cloudService = cloudService
	}
}

func WithPrometheusRegistry(registry *prometheus.Registry) RancherOption {
	return func(r *rancher) {
		r.prometheusRegistry = registry
	}
}

func NewRancher(rancherOptions ...RancherOption) (*rancher, error) {
	r := rancher{
		garbageMutex:   &sync.Mutex{},
		garbageObjects: make(map[string]*normanTypes.Resource),
	}

	for _, rancherOption := range rancherOptions {
		rancherOption(&r)
	}

	if r.clientOpts.TokenKey == "" {
		r.logger.Debug("using rancher credentials from kubernetes")

		kubeconfig, err := config.GetConfig()
		if err != nil {
			r.logger.Error("error getting kubeconfig", "err", err)
			return nil, err
		}

		r.logger.Debug("creating new controller client")

		controllerClient, err := client.New(kubeconfig, client.Options{})
		if err != nil {
			r.logger.Error("error creating controller client", "err", err)
			return nil, err
		}

		r.controllerClient = controllerClient
		if !r.clientOpts.Insecure {
			r.logger.Info("insecure trust not configured, getting issuer from kubernetes")

			caCerts, err := r.getInternalCACerts()
			if err != nil {
				r.logger.Error("error getting internal certificate authority", "err", err)
				return nil, err
			}

			r.clientOpts.CACerts = caCerts
		}

		r.logger.Info("token key not configured, getting secret from kubernetes")

		tokenKey, err := r.getTokenKeyOrBootstrap()
		if err != nil {
			r.logger.Error("error getting token key from kubernetes", "err", err)
			return nil, err
		}

		r.logger.Debug("got token key from kubernetes", "key", tokenKey)

		r.clientOpts.TokenKey = tokenKey
	}

	managementClient, err := managementv3.NewClient(r.clientOpts)
	if err != nil {
		return nil, err
	}
	r.managementClient = managementClient

	if r.prometheusRegistry != nil {
		r.logger.Debug("prometheus registry set, adding metrics")

		clusterMetric := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "dockyards_backend_cluster",
				ConstLabels: prometheus.Labels{
					"clusterservice": "rancher",
				},
			},
			[]string{
				"name",
				"organization_name",
				"state",
			},
		)

		r.clusterMetric = clusterMetric

		r.prometheusRegistry.MustRegister(r.clusterMetric)

	}

	return &r, err
}

func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(b int64) *int64 {
	return &b
}
