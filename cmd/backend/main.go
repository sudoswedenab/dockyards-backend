package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/handlers"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/cluster"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/jwt"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/user"
	"bitbucket.org/sudosweden/backend/api/v1/routes"
	_ "bitbucket.org/sudosweden/backend/docs"
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/rancher"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/exp/slog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	internal.LoadEnvVariables()
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
	return slog.New(handlerOptions.NewTextHandler(os.Stdout)), nil
}

//	@title			Themis API
//	@version		1.0
//	@description	This server.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Proprietary
//	@license.url	Copyright©

// @host		localhost:9000
// @BasePath	/v1/
func main() {
	var logLevel string
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.Parse()

	logger, err := newLogger(logLevel)
	if err != nil {
		fmt.Printf("error preparing logger: %s", err)
		os.Exit(1)
	}

	var db *gorm.DB
	var connectToDB func(*sync.WaitGroup)

	connectToDB = func(wg *sync.WaitGroup) {
		logger.Info("Trying to connect..")
		dsn := internal.DatabaseConf
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			logger.Info("Failed to connect to database, trying again..")
			time.Sleep(time.Second * 3)
			connectToDB(wg)
		} else {
			logger.Info("Success!")
			wg.Done()
		}
	}

	internal.WaitUntil(connectToDB)
	err = internal.SyncDataBase(db)
	if err != nil {
		logger.Error("Failed to initialize database", "err", err)
		os.Exit(1)
	}

	rancherService, err := rancher.NewRancher(internal.CattleBearerToken, internal.CattleUrl, logger)
	if err != nil {
		log.Fatal(err.Error())
	}

	logger.Info("rancher info", "url", internal.CattleUrl)

	err = rancherService.CreateClusterRole()
	if err != nil {
		log.Fatal(err.Error())
	}

	r := gin.Default()
	i := gin.Default()

	if internal.FlagUseCors {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "https://demo.k8s.dockyards.io"},
			AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	routes.RegisterRoutes(r, db, rancherService)
	handlers.RegisterRoutes(r, db, rancherService, logger)
	jwt.RegisterRoutes(r, db, rancherService)
	user.RegisterRoutes(r, db, rancherService)
	cluster.RegisterRoutes(r, rancherService)

	routes.RegisterRoutesInternal(i)

	if internal.FlagUseSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	go i.Run(":9001")
	r.Run()
}
