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
	"sync"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/handlers"
	"bitbucket.org/sudosweden/dockyards-backend/internal"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/cloudmock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/openstack"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/clustermock"
	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices/rancher"
	"bitbucket.org/sudosweden/dockyards-backend/internal/controller"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/metrics"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/jwt"
	openstackv1alpha1 "bitbucket.org/sudosweden/dockyards-openstack/api/v1alpha1"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-logr/logr/funcr"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func buildDataSourceName() string {
	conf := os.Getenv("DB_CONF")
	if conf != "" {
		return conf
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", host, port, user, password, name)
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
	var useInmemDb bool
	var trustInsecure bool
	var delGarbageInterval int
	var collectMetricsInterval int
	var ginMode string
	var cloudServiceFlag string
	var clusterServiceFlag string
	var insecureLogging bool
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.BoolVar(&useInmemDb, "use-inmem-db", false, "use in-memory database")
	flag.BoolVar(&trustInsecure, "trust-insecure", false, "trust all certs")
	flag.IntVar(&delGarbageInterval, "del-garbage-interval", 60, "delete garbage interval seconds")
	flag.IntVar(&collectMetricsInterval, "collect-metrics-interval", 30, "collect metrics interval seconds")
	flag.StringVar(&ginMode, "gin-mode", gin.DebugMode, "gin mode")
	flag.StringVar(&cloudServiceFlag, "cloud-service", "openstack", "cloud service")
	flag.StringVar(&clusterServiceFlag, "cluster-service", "rancher", "cluster service")
	flag.BoolVar(&insecureLogging, "insecure-logging", false, "insecure logging")
	flag.Parse()

	logger, err := newLogger(logLevel)
	if err != nil {
		fmt.Printf("error preparing logger: %s", err)
		os.Exit(1)
	}

	flag.Parse()

	var db *gorm.DB
	var connectToDB func(*sync.WaitGroup)

	gormLogger := loggers.NewGormSlogger(logger.With("orm", "gorm"))
	gormConfig := gorm.Config{
		Logger:         gormLogger,
		TranslateError: true,
	}

	if useInmemDb {
		db, err = gorm.Open(sqlite.Open(":memory:"), &gormConfig)
		if err != nil {
			logger.Error("error creating in-memory db", "err", err)
			os.Exit(1)
		}
	} else {
		dsn := buildDataSourceName()
		logger.Debug("database config", "dsn", dsn)
		connectToDB = func(wg *sync.WaitGroup) {
			logger.Info("Trying to connect..")
			db, err = gorm.Open(postgres.Open(dsn), &gormConfig)
			if err != nil {
				logger.Info("Failed to connect to database, trying again..")
				time.Sleep(time.Second * 3)
				connectToDB(wg)
			} else {
				fmt.Println("Success!")
				wg.Done()
			}
		}

		internal.WaitUntil(connectToDB)
	}

	logr := funcr.New(func(format, args string) { fmt.Println(format, args) }, funcr.Options{})
	ctrl.SetLogger(logr)

	kubeconfig, err := config.GetConfig()
	if err != nil {
		logger.Error("error getting kubeconfig", "err", err)

		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
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

	for _, object := range []client.Object{&v1alpha1.User{}, &v1alpha1.Cluster{}, &v1alpha1.NodePool{}, &v1alpha1.Node{}, &v1alpha1.Deployment{}} {
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

	for _, object := range []client.Object{&v1alpha1.NodePool{}, &v1alpha1.Node{}, &v1alpha1.Deployment{}} {
		err = manager.GetFieldIndexer().IndexField(ctx, object, index.OwnerRefsIndexKey, index.OwnerRefsIndexer)
		if err != nil {
			logger.Error("error addming owner refs indexer to manager", "err", err)

			os.Exit(1)
		}
	}

	var cloudService cloudservices.CloudService
	switch cloudServiceFlag {
	case "openstack":
		openStackOptions := []openstack.OpenStackOption{
			openstack.WithLogger(logger.With("cloudservice", "openstack")),
			openstack.WithDatabase(db),
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
			rancher.WithPrometheusRegistry(registry),
		}

		clusterService, err = rancher.NewRancher(rancherOptions...)
		if err != nil {
			log.Fatal(err.Error())
		}
	case "clustermock":
		clusterService = clustermock.NewMockClusterService()
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
				clusterService.DeleteGarbage()
				cloudService.DeleteGarbage()
			}
		}
	}()

	prometheusMetricsOptions := []metrics.PrometheusMetricsOption{
		metrics.WithLogger(logger),
		metrics.WithDatabase(db),
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
		clusterService.CollectMetrics()

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				prometheusMetrics.CollectMetrics()
				clusterService.CollectMetrics()
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
		handlers.WithCloudService(cloudService),
		handlers.WithClusterService(clusterService),
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

	err = controller.NewOrganizationController(manager, logger.With("controller", "organization"))
	if err != nil {
		logger.Error("error creating new organization controller", "err", err)

		os.Exit(1)
	}

	err = controller.NewClusterController(manager, clusterService, logger.With("controller", "cluster"))
	if err != nil {
		logger.Error("error creating new cluster controller", "err", err)

		os.Exit(1)
	}

	err = controller.NewNodePoolController(manager, clusterService, logger.With("controller", "nodepool"))
	if err != nil {
		logger.Error("error creating new cluster controller", "err", err)

		os.Exit(1)
	}

	err = controller.NewNodeController(manager, clusterService, logger.With("controller", "node"))
	if err != nil {
		logger.Error("error creating new node controller", "err", err)

		os.Exit(1)
	}

	err = manager.Start(context.Background())
	if err != nil {
		logger.Error("error starting manager", "err", err)

		os.Exit(1)
	}
}
