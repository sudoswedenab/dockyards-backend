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
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
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
	pflag.StringSliceVar(&allowOrigins, "allow-origin", []string{"http://localhost"}, "allow origin")
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

	for _, object := range []client.Object{&v1alpha1.Deployment{}, &v1alpha1.App{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, v1alpha1index.UIDIndexKey, v1alpha1index.UIDIndexer)
		if err != nil {
			logger.Error("error adding uid indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	for _, object := range []client.Object{&dockyardsv1.User{}, &dockyardsv1.Cluster{}, &dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &corev1.Secret{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, index.UIDField, index.ByUID)
		if err != nil {
			logger.Error("error adding uid indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	err = manager.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Organization{}, index.MemberRefsIndexKey, index.MemberRefsIndexer)
	if err != nil {
		logger.Error("error adding member refs indexer to manager", "err", err)

		os.Exit(1)
	}

	for _, object := range []client.Object{&v1alpha1.NodePool{}, &v1alpha1.Node{}, &v1alpha1.Deployment{}, &v1alpha1.Cluster{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, v1alpha1index.OwnerRefsIndexKey, v1alpha1index.OwnerRefsIndexer)
		if err != nil {
			logger.Error("error adding owner refs indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	for _, object := range []client.Object{&dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &dockyardsv1.Cluster{}} {
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

		prometheusMetrics.CollectMetrics()

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				prometheusMetrics.CollectMetrics()
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

	accessKey, refreshKey, err := jwt.GetOrGenerateKeys(ctx, controllerClient, logger)
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

	privateRouter := chi.NewRouter()
	privateRouter.Use(middleware.Logger)

	promHandlerOpts := promhttp.HandlerOpts{
		Registry: registry,
	}
	promHandler := promhttp.HandlerFor(registry, promHandlerOpts)

	privateRouter.Handle("/metrics", promHandler)
	privateRouter.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {})

	privateServer := &http.Server{
		Handler: privateRouter,
		Addr:    ":9001",
	}

	go privateServer.ListenAndServe()
	go r.Run(":9000")

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

	if enableWebhooks {
		err = (&v1alpha1.Organization{}).SetupWebhookWithManager(manager)
		if err != nil {
			logger.Error("error creating organization webhook", "err", err)

			os.Exit(1)
		}

		err = (&dockyardsv1.Organization{}).SetupWebhookWithManager(manager)
		if err != nil {
			logger.Error("error creating organization webhook", "err", err)

			os.Exit(1)
		}

		err = (&webhooks.DockyardsNodePool{}).SetupWebhookWithManager(manager)
		if err != nil {
			logger.Error("error creating nodepool webhook", "err", err)

			os.Exit(1)
		}

		err = (&webhooks.DockyardsCluster{}).SetupWebhookWithManager(manager)
		if err != nil {
			logger.Error("error creating cluster webhook", "err", err)

			os.Exit(1)
		}

		err = (&webhooks.DockyardsOrganization{}).SetupWebhookWithManager(manager)
		if err != nil {
			logger.Error("error creating organization webhook", "err", err)

			os.Exit(1)
		}
	}

	err = manager.Start(context.Background())
	if err != nil {
		logger.Error("error starting manager", "err", err)

		os.Exit(1)
	}
}
