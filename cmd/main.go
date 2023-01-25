package main

import (
	"Backend/api/v1/routes"
	_ "Backend/docs"
	"Backend/internal"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func init() {
	internal.LoadEnvVariables()
	internal.WaitUntil(internal.ConnectToDB)
	internal.SyncDataBase()
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
	r := gin.Default()
	useCors, err := strconv.ParseBool(os.Getenv("FLAG_USE_CORS"))
	if err != nil {
		fmt.Printf("error parsing: %s", err)
		return
	}

	if useCors {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000"},
			AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
			AllowHeaders:     []string{"Origin", "Content-Type"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			AllowOriginFunc: func(origin string) bool {
				return origin == "http://localhost:3000"
			},
			MaxAge: 12 * time.Hour,
		}))
	}

	routes.RegisterRoutes(r)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run()
}
