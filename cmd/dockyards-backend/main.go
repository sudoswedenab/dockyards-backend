package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/handlers"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/handlers/user"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/routes"
	"bitbucket.org/sudosweden/dockyards-backend/internal"
	"bitbucket.org/sudosweden/dockyards-backend/internal/cloudservices/openstack"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/metrics"
	"bitbucket.org/sudosweden/dockyards-backend/internal/rancher"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	jwtAccessTokenSecret  string
	jwtRefreshTokenSecret string
	cattleURL             string
	cattleBearerToken     string
	flagUseCors           = false
	flagServerCookie      = false
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

	flagUseCors, err = strconv.ParseBool(os.Getenv("FLAG_USE_CORS"))
	if err != nil {
		fmt.Printf("error parsing FLAG_USE_CORS: %s", err)
	}
	flagServerCookie, err = strconv.ParseBool(os.Getenv("FLAG_SET_SERVER_COOKIE"))
	if err != nil {
		fmt.Printf("error parsing FLAG_SET_SERVER_COOKIE: %s", err)
	}

	jwtAccessTokenSecret = os.Getenv("JWT_ACCESS_TOKEN_SECRET")
	jwtRefreshTokenSecret = os.Getenv("JWT_REFRESH_TOKEN_SECRET")
	cattleURL = os.Getenv("CATTLE_URL")
	cattleBearerToken = os.Getenv("CATTLE_BEARER_TOKEN")
}

func main() {
	var logLevel string
	var useInmemDb bool
	var trustInsecure bool
	var delGarbageInterval int
	var collectMetricsInterval int
	var ginMode string
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.BoolVar(&useInmemDb, "use-inmem-db", false, "use in-memory database")
	flag.BoolVar(&trustInsecure, "trust-insecure", false, "trust all certs")
	flag.IntVar(&delGarbageInterval, "del-garbage-interval", 60, "delete garbage interval seconds")
	flag.IntVar(&collectMetricsInterval, "collect-metrics-interval", 30, "collect metrics interval seconds")
	flag.StringVar(&ginMode, "gin-mode", gin.DebugMode, "gin mode")
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
		Logger: gormLogger,
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

	err = internal.SyncDataBase(db)
	if err != nil {
		logger.Error("Failed to initialize database", "err", err)
		os.Exit(1)
	}

	err = openstack.SyncDatabase(db)
	if err != nil {
		logger.Error("error syncing database with openstack models", "err", err)
		os.Exit(1)
	}

	openStackOptions := []openstack.OpenStackOption{
		openstack.WithLogger(logger.With("cloudservice", "openstack")),
		openstack.WithDatabase(db),
		openstack.WithCloudsYAML("openstack"),
	}

	cloudService, err := openstack.NewOpenStackService(openStackOptions...)
	if err != nil {
		logger.Error("error creating new openstack cloud service", "err", err)
		os.Exit(1)
	}

	registry := prometheus.NewRegistry()

	rancherOptions := []rancher.RancherOption{
		rancher.WithRancherClientOpts(cattleURL, cattleBearerToken, trustInsecure),
		rancher.WithLogger(logger.With("clusterservice", "rancher")),
		rancher.WithCloudService(cloudService),
		rancher.WithPrometheusRegistry(registry),
	}

	rancherService, err := rancher.NewRancher(rancherOptions...)
	if err != nil {
		log.Fatal(err.Error())
	}

	go func() {
		interval := time.Second * time.Duration(delGarbageInterval)

		logger.Debug("creating garbage deletion goroutine", "interval", interval)

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				rancherService.DeleteGarbage()
			}
		}

	}()

	logger.Info("rancher info", "url", cattleURL)

	prometheusMetricsOptions := []metrics.PrometheusMetricsOption{
		metrics.WithLogger(logger),
		metrics.WithDatabase(db),
		metrics.WithPrometheusRegistry(registry),
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
		rancherService.CollectMetrics()

		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				prometheusMetrics.CollectMetrics()
				rancherService.CollectMetrics()
			}
		}
	}()

	logger.Debug("setting gin mode", "mode", ginMode)

	gin.SetMode(ginMode)

	r := gin.Default()
	i := gin.Default()

	if flagUseCors {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "https://demo.k8s.dockyards.io"},
			AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	handlerOptions := []handlers.HandlerOption{
		handlers.WithJWTAccessTokens(jwtAccessTokenSecret, jwtRefreshTokenSecret),
		handlers.WithCloudService(cloudService),
	}

	routes.RegisterRoutes(r, db, rancherService)
	err = handlers.RegisterRoutes(r, db, rancherService, logger, flagServerCookie, handlerOptions...)
	if err != nil {
		logger.Error("error registering handler routes", "err", err)
		os.Exit(1)
	}

	user.RegisterRoutes(r, db)

	sudoHandlerOptions := []handlers.SudoHandlerOption{
		handlers.WithSudoClusterService(rancherService),
		handlers.WithSudoLogger(logger),
		handlers.WithSudoGormDB(db),
		handlers.WithSudoPrometheusRegistry(registry),
	}

	handlers.RegisterSudoRoutes(i, sudoHandlerOptions...)

	go i.Run(":9001")
	r.Run()
}
