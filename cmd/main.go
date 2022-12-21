package main

import (
	"Backend/api/v1/routes"
	"Backend/internal"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func init() {
	internal.LoadEnvVariables()
	internal.ConnectToDB()
	internal.SyncDataBase()
}

func main() {

	r := gin.Default()

	routes.RegisterRoutes(r)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000/"},
		AllowMethods:     []string{"POST", "PUT", "GET", "DELETE"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://localhost:3000/"
		},
		MaxAge: 12 * time.Hour,
	}))

	r.Run()
}
