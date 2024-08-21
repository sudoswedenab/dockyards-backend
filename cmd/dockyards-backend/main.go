package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/handlers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/controller"
	"bitbucket.org/sudosweden/dockyards-backend/internal/metrics"
	"bitbucket.org/sudosweden/dockyards-backend/internal/webhooks"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	v1alpha1index "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
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

func setupWebhooks(mgr ctrl.Manager) error {
	err := (&v1alpha1.Organization{}).SetupWebhookWithManager(mgr)
	if err != nil {
		//logger.Error("error creating organization webhook", "err", err)
		return err
	}

	err = (&dockyardsv1.Organization{}).SetupWebhookWithManager(mgr)
	if err != nil {
		//logger.Error("error creating organization webhook", "err", err)

		return err
	}

	err = (&webhooks.DockyardsNodePool{}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&webhooks.DockyardsCluster{}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	err = (&webhooks.DockyardsOrganization{}).SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var logLevel string
	var collectMetricsInterval int
	var ginMode string
	var enableWebhooks bool
	var metricsBindAddress string
	var allowOrigins []string
	var dockyardsNamespace string
	pflag.StringVar(&logLevel, "log-level", "info", "log level")
	pflag.IntVar(&collectMetricsInterval, "collect-metrics-interval", 30, "collect metrics interval seconds")
	pflag.StringVar(&ginMode, "gin-mode", gin.DebugMode, "gin mode")
	pflag.BoolVar(&enableWebhooks, "enable-webhooks", false, "enable webhooks")
	pflag.StringVar(&metricsBindAddress, "metrics-bind-address", "0", "metrics bind address")
	pflag.StringSliceVar(&allowOrigins, "allow-origin", []string{"http://localhost", "http://localhost:8000"}, "allow origin")
	pflag.StringVar(&dockyardsNamespace, "dockyards-namespace", "dockyards", "dockyards namespace")
	pflag.Parse()

	logger, err := newLogger(logLevel)
	if err != nil {
		fmt.Printf("error preparing logger: %s", err)
		os.Exit(1)
	}

	slogr := logr.FromSlogHandler(logger.Handler())
	ctrl.SetLogger(slogr)

	kubeconfig, err := config.GetConfig()
	if err != nil {
		logger.Error("error getting kubeconfig", "err", err)

		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = authorizationv1.AddToScheme(scheme)

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

	manager, err := ctrl.NewManager(kubeconfig, managerOptions)
	if err != nil {
		logger.Error("error creating manager", "err", err)

		os.Exit(1)
	}

	ctx := context.Background()

	err = manager.GetFieldIndexer().IndexField(ctx, &dockyardsv1.User{}, index.EmailField, index.ByEmail)
	if err != nil {
		logger.Error("error adding email indexer to manager", "err", err)

		os.Exit(1)
	}

	for _, object := range []client.Object{&v1alpha1.App{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, v1alpha1index.UIDIndexKey, v1alpha1index.UIDIndexer)
		if err != nil {
			logger.Error("error adding uid indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	for _, object := range []client.Object{&dockyardsv1.User{}, &dockyardsv1.Cluster{}, &dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &corev1.Secret{}, &dockyardsv1.Deployment{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, index.UIDField, index.ByUID)
		if err != nil {
			logger.Error("error adding uid indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	err = manager.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Organization{}, index.MemberReferencesField, index.ByMemberReferences)
	if err != nil {
		logger.Error("error adding member refs indexer to manager", "err", err)

		os.Exit(1)
	}

	for _, object := range []client.Object{&dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &dockyardsv1.Cluster{}, &dockyardsv1.Deployment{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, index.OwnerReferencesField, index.ByOwnerReferences)
		if err != nil {
			logger.Error("error adding owner refs indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	err = manager.GetFieldIndexer().IndexField(ctx, &corev1.Secret{}, index.SecretTypeField, index.BySecretType)
	if err != nil {
		logger.Error("error adding secret type indexer to manager", "err", err)

		os.Exit(1)
	}

	registry := prometheus.NewRegistry()

	prometheusMetricsOptions := []metrics.PrometheusMetricsOption{
		metrics.WithLogger(logger),
		metrics.WithPrometheusRegistry(registry),
		metrics.WithManager(manager),
	}

	prometheusMetrics, err := metrics.NewPrometheusMetrics(prometheusMetricsOptions...)
	if err != nil {
		logger.Error("error creating new prometheus metrics", "err", err)
		os.Exit(1)
	}

	go func() {
		interval := time.Second * time.Duration(collectMetricsInterval)

		logger.Debug("creating prometheus metrics goroutine", "interval", interval)

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

	logger.Debug("setting gin mode", "mode", ginMode)

	gin.SetMode(ginMode)

	r := gin.Default()

	logger.Debug("configuring cors middleware", "origins", allowOrigins)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	accessKey, refreshKey, err := jwt.GetOrGenerateKeys(ctx, controllerClient)
	if err != nil {
		logger.Error("error getting private keys for jwt", "err", err)

		os.Exit(1)
	}

	handlerOptions := []handlers.HandlerOption{
		handlers.WithManager(manager),
		handlers.WithNamespace(dockyardsNamespace),
		handlers.WithJWTPrivateKeys(accessKey, refreshKey),
	}

	err = handlers.RegisterRoutes(r, logger, handlerOptions...)
	if err != nil {
		logger.Error("error registering handler routes", "err", err)
		os.Exit(1)
	}

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

	go func() {
		err := r.Run(":9000")
		if err != nil {
			logger.Error("error running public server", "err", err)

			os.Exit(1)
		}
	}()

	err = (&controller.OrganizationReconciler{
		Client: manager.GetClient(),
	}).SetupWithManager(manager)
	if err != nil {
		logger.Error("error creating new organization reconciler", "err", err)

		os.Exit(1)
	}

	err = (&controller.FeatureReconciler{
		Client:             manager.GetClient(),
		DockyardsNamespace: dockyardsNamespace,
	}).SetupWithManager(manager)
	if err != nil {
		logger.Error("error creating new feature reconciler", "err", err)

		os.Exit(1)
	}

	err = (&controller.ClusterReconciler{
		Client:             manager.GetClient(),
		DockyardsNamespace: dockyardsNamespace,
	}).SetupWithManager(manager)
	if err != nil {
		logger.Error("error creating new cluster reconciler", "err", err)

		os.Exit(1)
	}

	if enableWebhooks {
		err := setupWebhooks(manager)
		if err != nil {
			logger.Error("error creating webhooks", "err", err)

			os.Exit(1)
		}
	}

	err = manager.Start(context.Background())
	if err != nil {
		logger.Error("error starting manager", "err", err)

		os.Exit(1)
	}
}
