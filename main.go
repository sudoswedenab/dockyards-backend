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

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/spf13/pflag"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/handlers"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v2"
	"github.com/sudoswedenab/dockyards-backend/internal/controller"
	"github.com/sudoswedenab/dockyards-backend/internal/metrics"
	"github.com/sudoswedenab/dockyards-backend/internal/webhooks"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/jwt"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func newLogger(logLevel string) (*slog.Logger, error) {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log level %s", logLevel)
	}

	handlerOptions := slog.HandlerOptions{
		Level: level,
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &handlerOptions)), nil
}

func setupWebhooks(mgr ctrl.Manager, allowedDomains []string) error {
	err := (&webhooks.DockyardsNodePool{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&webhooks.DockyardsCluster{}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&webhooks.DockyardsOrganization{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&webhooks.DockyardsUser{
		Client:         mgr.GetClient(),
		AllowedDomains: allowedDomains,
	}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&webhooks.DockyardsInvitation{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&dockyardsv1.Organization{}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var logLevel string
	var configMap string
	var collectMetricsInterval int
	var enableWebhooks bool
	var metricsBindAddress string
	var allowedOrigins []string
	var dockyardsNamespace string
	var allowedDomains []string
	pflag.StringVar(&logLevel, "log-level", "info", "log level")
	pflag.StringVar(&configMap, "config-map", "dockyards-system", "ConfigMap name")
	pflag.IntVar(&collectMetricsInterval, "collect-metrics-interval", 30, "collect metrics interval seconds")
	pflag.BoolVar(&enableWebhooks, "enable-webhooks", false, "enable webhooks")
	pflag.StringVar(&metricsBindAddress, "metrics-bind-address", "0", "metrics bind address")
	pflag.StringSliceVar(&allowedOrigins, "allow-origin", []string{"http://localhost", "http://localhost:8000"}, "allow origin")
	pflag.StringVar(&dockyardsNamespace, "dockyards-namespace", "dockyards-system", "dockyards namespace")
	pflag.StringSliceVar(&allowedDomains, "allow-domain", nil, "allow domain")
	pflag.Parse()

	logger, err := newLogger(logLevel)
	if err != nil {
		fmt.Printf("error preparing logger: %s", err)
		os.Exit(1)
	}

	slogr := logr.FromSlogHandler(logger.Handler())
	ctrl.SetLogger(slogr)

	wd, err := os.Getwd()
	if err != nil {
		logger.Error("error getting working directory", "err", err)

		os.Exit(1)
	}

	logger.Debug("process info", "wd", wd, "uid", os.Getuid(), "pid", os.Getpid(), "namespace", dockyardsNamespace)

	kubeconfig, err := config.GetConfig()
	if err != nil {
		logger.Error("error getting kubeconfig", "err", err)

		os.Exit(1)
	}

	scheme := runtime.NewScheme()

	_ = authorizationv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("error creating new controller client", "err", err)

		os.Exit(1)
	}

	managerOptions := ctrl.Options{
		Scheme:                 scheme,
		Client:                 client.Options{},
		HealthProbeBindAddress: "0",
		Metrics: metricsserver.Options{
			BindAddress: metricsBindAddress,
		},
	}

	mgr, err := ctrl.NewManager(kubeconfig, managerOptions)
	if err != nil {
		logger.Error("error creating manager", "err", err)

		os.Exit(1)
	}

	ctx := context.Background()

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		logger.Error("error adding default indexes", "err", err)

		os.Exit(1)
	}

	registry := prometheus.NewRegistry()

	prometheusMetricsOptions := []metrics.PrometheusMetricsOption{
		metrics.WithLogger(logger),
		metrics.WithPrometheusRegistry(registry),
		metrics.WithManager(mgr),
	}

	prometheusMetrics, err := metrics.NewPrometheusMetrics(prometheusMetricsOptions...)
	if err != nil {
		logger.Error("error creating new prometheus metrics", "err", err)
		os.Exit(1)
	}

	go func() {
		synced := mgr.GetCache().WaitForCacheSync(ctx)
		if !synced {
			logger.Warn("collecting metrics before cache is synced")
		}

		interval := time.Second * time.Duration(collectMetricsInterval)

		err := prometheusMetrics.CollectMetrics()
		if err != nil {
			logger.Error("error collecting prometheus metrics", "err", err)
		}

		ticker := time.NewTicker(interval)
		for range ticker.C {
			err := prometheusMetrics.CollectMetrics()
			if err != nil {
				logger.Error("error collecting prometheus metrics", "err", err)
			}
		}
	}()

	accessKey, refreshKey, err := jwt.GetOrGenerateKeys(ctx, controllerClient, dockyardsNamespace)
	if err != nil {
		logger.Error("error getting private keys for jwt", "err", err)

		os.Exit(1)
	}

	handlerOptions := []handlers.HandlerOption{
		handlers.WithManager(mgr),
		handlers.WithNamespace(dockyardsNamespace),
		handlers.WithJWTPrivateKeys(accessKey, refreshKey),
		handlers.WithLogger(logger),
	}

	publicMux := http.NewServeMux()

	err = handlers.RegisterRoutes(publicMux, handlerOptions...)
	if err != nil {
		logger.Error("error registering handler routes", "err", err)
		os.Exit(1)
	}

	corsOptions := cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodPost, http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Origin"},
		AllowCredentials: true,
		ExposedHeaders:   []string{"Content-Length"},
	}

	corsHandler := cors.New(corsOptions)

	publicHandler := corsHandler.Handler(publicMux)

	v2API := v2.NewAPI(mgr, &accessKey.PublicKey)
	v2API.RegisterRoutes(publicMux)

	publicServer := &http.Server{
		Handler: publicHandler,
		Addr:    ":9000",
	}

	go func() {
		err := publicServer.ListenAndServe()
		if err != nil {
			logger.Error("error running public server", "err", err)

			os.Exit(1)
		}
	}()

	privateMux := http.NewServeMux()

	promHandlerOpts := promhttp.HandlerOpts{
		Registry: registry,
	}
	promHandler := promhttp.HandlerFor(registry, promHandlerOpts)

	privateMux.Handle("/metrics", promHandler)
	privateMux.HandleFunc("GET /healthz", func(_ http.ResponseWriter, _ *http.Request) {})

	privateServer := &http.Server{
		Handler: privateMux,
		Addr:    ":9001",
	}

	go func() {
		err := privateServer.ListenAndServe()
		if err != nil {
			logger.Error("error running private server", "err", err)

			os.Exit(1)
		}
	}()

	err = (&controller.OrganizationReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		logger.Error("error creating new organization reconciler", "err", err)

		os.Exit(1)
	}

	err = (&controller.ClusterReconciler{
		Client:             mgr.GetClient(),
		DockyardsNamespace: dockyardsNamespace,
	}).SetupWithManager(mgr)
	if err != nil {
		logger.Error("error creating new cluster reconciler", "err", err)

		os.Exit(1)
	}

	err = (&controller.WorkloadReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		logger.Error("error creating new workload reconciler", "err", err)

		os.Exit(1)
	}

	dockyardsConfig, err := dyconfig.GetConfig(ctx, controllerClient, configMap, dockyardsNamespace)
	if err != nil {
		logger.Error("error loading config map", "err", err)

		os.Exit(1)
	}

	err = (&controller.UserReconciler{
		Client:          mgr.GetClient(),
		DockyardsConfig: dockyardsConfig,
	}).SetupWithManager(mgr)
	if err != nil {
		logger.Error("error creating new verificationrequest reconciler", "err", err)

		os.Exit(1)
	}

	err = (&controller.InvitationReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManger(mgr)
	if err != nil {
		logger.Error("error creating new invitation reconciler", "err", err)

		os.Exit(1)
	}

	if enableWebhooks {
		logger.Info("enabling webhooks", "domains", allowedDomains)

		err := setupWebhooks(mgr, allowedDomains)
		if err != nil {
			logger.Error("error creating webhooks", "err", err)

			os.Exit(1)
		}
	}

	err = authorization.ReconcileGlobalAuthorization(ctx, controllerClient)
	if err != nil {
		logger.Error("error reconciling global authorization", "err", err)

		os.Exit(1)
	}

	err = mgr.Start(context.Background())
	if err != nil {
		logger.Error("error starting manager", "err", err)

		os.Exit(1)
	}
}
