package main

import (
	"Backend/api/v1/routes"
	"Backend/internal"

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

	r.Run()
}
