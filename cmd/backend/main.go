package main

import (
	"fmt"
	"sync"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/handlers"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/jwt"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/user"
	"bitbucket.org/sudosweden/backend/api/v1/routes"
	_ "bitbucket.org/sudosweden/backend/docs"
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/rancher"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func init() {
	internal.LoadEnvVariables()
	internal.CreateClusterRole()
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
	var db *gorm.DB
	var connectToDB func(*sync.WaitGroup)
	var err error

	connectToDB = func(wg *sync.WaitGroup) {
		fmt.Println("Trying to connect..")
		dsn := internal.DatabaseConf
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			fmt.Println("Failed to connect to database, trying again..")
			time.Sleep(time.Second * 3)
			connectToDB(wg)
		} else {
			fmt.Println("Success!")
			wg.Done()
		}
	}

	internal.WaitUntil(connectToDB)
	internal.SyncDataBase(db)

	rancherService := rancher.Rancher{
		BearerToken: internal.CattleBearerToken,
		Url:         internal.CattleUrl,
	}

	r := gin.Default()

	if internal.FlagUseCors {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "https://demo.k8s.dockyards.io/"},
			AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	routes.RegisterRoutes(r, db, &rancherService)
	handlers.RegisterRoutes(r, db, &rancherService)
	jwt.RegisterRoutes(r, db, &rancherService)
	user.RegisterRoutes(r, db, &rancherService)

	if internal.FlagUseSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	r.Run()
}
