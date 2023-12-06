package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/handlers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/cloudmock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/openstack"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/rancher"
	"bitbucket.org/sudosweden/dockyards-backend/internal/controller"
	"bitbucket.org/sudosweden/dockyards-backend/internal/metrics"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/jwt"
	openstackv1alpha1 "bitbucket.org/sudosweden/dockyards-openstack/api/v1alpha1"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-logr/logr/slogr"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	cattleURL         string
	cattleBearerToken string
	corsAllowOrigins  string
)

func init() {
	loadEnvVariables()
}

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

func loadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		log.Println("could not load .env file")
	}

	cattleURL = os.Getenv("CATTLE_URL")
	cattleBearerToken = os.Getenv("CATTLE_BEARER_TOKEN")
	corsAllowOrigins = os.Getenv("CORS_ALLOW_ORIGINS")
}

func main() {
	var logLevel string
	var trustInsecure bool
	var delGarbageInterval int
	var collectMetricsInterval int
	var ginMode string
	var cloudServiceFlag string
	var clusterServiceFlag string
	var insecureLogging bool
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.BoolVar(&trustInsecure, "trust-insecure", false, "trust all certs")
	flag.IntVar(&delGarbageInterval, "del-garbage-interval", 60, "delete garbage interval seconds")
	flag.IntVar(&collectMetricsInterval, "collect-metrics-interval", 30, "collect metrics interval seconds")
	flag.StringVar(&ginMode, "gin-mode", gin.DebugMode, "gin mode")
	flag.StringVar(&cloudServiceFlag, "cloud-service", "none", "cloud service")
	flag.StringVar(&clusterServiceFlag, "cluster-service", "none", "cluster service")
	flag.BoolVar(&insecureLogging, "insecure-logging", false, "insecure logging")
	flag.Parse()

	logger, err := newLogger(logLevel)
	if err != nil {
		fmt.Printf("error preparing logger: %s", err)
		os.Exit(1)
	}

	logr := slogr.NewLogr(logger.Handler())
	ctrl.SetLogger(logr)

	kubeconfig, err := config.GetConfig()
	if err != nil {
		logger.Error("error getting kubeconfig", "err", err)

		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	v1alpha2.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	openstackv1alpha1.AddToScheme(scheme)

	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("error creating new controller client", "err", err)

		os.Exit(1)
	}

	managerOptions := ctrl.Options{
		Scheme: scheme,
		Client: client.Options{},
	}

	manager, err := ctrl.NewManager(kubeconfig, managerOptions)
	if err != nil {
		logger.Error("error creating manager", "err", err)

		os.Exit(1)
	}

	ctx := context.Background()

	err = manager.GetFieldIndexer().IndexField(ctx, &v1alpha1.User{}, index.EmailIndexKey, index.EmailIndexer)
	if err != nil {
		logger.Error("error adding email indexer to manager", "err", err)

		os.Exit(1)
	}

	for _, object := range []client.Object{&v1alpha1.User{}, &v1alpha1.Cluster{}, &v1alpha1.NodePool{}, &v1alpha1.Node{}, &v1alpha1.Deployment{}, &v1alpha1.App{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, index.UIDIndexKey, index.UIDIndexer)
		if err != nil {
			logger.Error("error adding uid indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	err = manager.GetFieldIndexer().IndexField(ctx, &v1alpha1.Organization{}, index.MemberRefsIndexKey, index.MemberRefsIndexer)
	if err != nil {
		logger.Error("error adding member refs indexer to manager", "err", err)

		os.Exit(1)
	}

	for _, object := range []client.Object{&v1alpha1.NodePool{}, &v1alpha1.Node{}, &v1alpha1.Deployment{}, &v1alpha1.Cluster{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, index.OwnerRefsIndexKey, index.OwnerRefsIndexer)
		if err != nil {
			logger.Error("error adding owner refs indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	err = manager.GetFieldIndexer().IndexField(ctx, &corev1.Secret{}, index.SecretTypeIndexKey, index.SecretTypeIndexer)
	if err != nil {
		logger.Error("error adding secret type indexer to manager", "err", err)

		os.Exit(1)
	}

	var cloudService cloudservices.CloudService
	switch cloudServiceFlag {
	case "openstack":
		openStackOptions := []openstack.OpenStackOption{
			openstack.WithLogger(logger.With("cloudservice", "openstack")),
			openstack.WithCloudsYAML("openstack"),
			openstack.WithInsecureLogging(insecureLogging),
			openstack.WithManager(manager),
		}

		cloudService, err = openstack.NewOpenStackService(openStackOptions...)
		if err != nil {
			logger.Error("error creating new openstack cloud service", "err", err)
			os.Exit(1)
		}
	case "cloudmock":
		cloudService = cloudmock.NewMockCloudService()
	case "none":
		logger.Info("not using a cloud service")
	default:
		logger.Error("unsupported cloud service", "service", cloudServiceFlag)

		os.Exit(1)
	}

	registry := prometheus.NewRegistry()

	var clusterService clusterservices.ClusterService
	switch clusterServiceFlag {
	case "rancher":
		rancherOptions := []rancher.RancherOption{
			rancher.WithRancherClientOpts(cattleURL, cattleBearerToken, trustInsecure),
			rancher.WithLogger(logger.With("clusterservice", "rancher")),
			rancher.WithCloudService(cloudService),
		}

		clusterService, err = rancher.NewRancher(rancherOptions...)
		if err != nil {
			log.Fatal(err.Error())
		}
	case "clustermock":
		clusterService = clustermock.NewMockClusterService()
	case "none":
		logger.Info("not using a cluster service")
	default:
		logger.Error("unsupported cluster service", "service", clusterServiceFlag)

		os.Exit(1)
	}

	go func() {
		interval := time.Second * time.Duration(delGarbageInterval)

		logger.Debug("creating garbage deletion goroutine", "interval", interval)

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				if clusterServiceFlag != "none" {
					clusterService.DeleteGarbage()
				}

				if cloudServiceFlag != "none" {
					cloudService.DeleteGarbage()
				}
			}
		}
	}()

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

	if corsAllowOrigins != "" {
		allowOrigins := strings.Split(corsAllowOrigins, ",")

		logger.Debug("configuring cors middleware", "origins", allowOrigins)

		r.Use(cors.New(cors.Config{
			AllowOrigins:     allowOrigins,
			AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	accessKey, refreshKey, err := jwt.GetOrGenerateKeys(ctx, controllerClient, logger)
	if err != nil {
		logger.Error("error getting private keys for jwt", "err", err)

		os.Exit(1)
	}

	handlerOptions := []handlers.HandlerOption{
		handlers.WithManager(manager),
		handlers.WithNamespace("dockyards"),
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
		Logger: logger.With("reconciler", "organization"),
	}).SetupWithManager(manager)
	if err != nil {
		logger.Error("error creating new organization reconciler", "err", err)

		os.Exit(1)
	}

	err = (&v1alpha1.Organization{}).SetupWebhookWithManager(manager)
	if err != nil {
		logger.Error("error creating organization webhook", "err", err)

		os.Exit(1)
	}

	err = (&v1alpha2.Organization{}).SetupWebhookWithManager(manager)
	if err != nil {
		logger.Error("error creating organization webhook", "err", err)

		os.Exit(1)
	}

	if clusterServiceFlag != "none" {
		err = (&controller.ClusterReconciler{
			Client:         manager.GetClient(),
			Logger:         logger.With("controller", "cluster"),
			ClusterService: clusterService,
		}).SetupWithManager(manager)
		if err != nil {
			logger.Error("error creating new cluster controller", "err", err)

			os.Exit(1)
		}

		err = (&controller.NodePoolReconciler{
			Client:         manager.GetClient(),
			Logger:         logger.With("controller", "nodepool"),
			ClusterService: clusterService,
		}).SetupWithManager(manager)
		if err != nil {
			logger.Error("error creating node pool controller", "err", err)

			os.Exit(1)
		}

		err = (&controller.NodeReconciler{
			Client:         manager.GetClient(),
			Logger:         logger.With("controller", "node"),
			ClusterService: clusterService,
		}).SetupWithManager(manager)
		if err != nil {
			logger.Error("error creating new node controller", "err", err)

			os.Exit(1)
		}

		err = (&controller.ReleaseReconciler{
			Client:         manager.GetClient(),
			Logger:         logger,
			ClusterService: clusterService,
		}).SetupWithManager(manager)
		if err != nil {
			logger.Error("error creating release controller", "err", err)

			os.Exit(1)
		}
	}

	err = manager.Start(context.Background())
	if err != nil {
		logger.Error("error starting manager", "err", err)

		os.Exit(1)
	}
}
