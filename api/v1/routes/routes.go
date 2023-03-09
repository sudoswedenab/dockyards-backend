package routes

import (
	"bitbucket.org/sudosweden/backend/api/v1/handlers"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/cluster"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/genbody"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/genkubeconfig"
	"bitbucket.org/sudosweden/backend/api/v1/handlers/user"
	"bitbucket.org/sudosweden/backend/api/v1/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Slash API",
		})
	})

	v1 := r.Group("/v1")

	v1.POST("/logout", func(c *gin.Context) {
		handlers.Logout(c)
	})

	v1.GET("/readcookie", func(c *gin.Context) {
		handlers.ReadCookie(c)
	})

	v1.GET("/mapcluster", func(c *gin.Context) {
		cluster.MapGetClusters(c)
	})

	v1.POST("/createcluster", func(c *gin.Context) {
		id, name, err := cluster.CreatedCluster(c)
		cluster.CreatedNodePool(c, id, name, err)
	})
	v1.DELETE("/deletecluster/:id", func(c *gin.Context) {
		cluster.DeleteCluster(c)
	})
	///
	// Admin Routes
	v1Admin := v1.Group("/admin", func(c *gin.Context) {
		// Handles errors
		middleware.RequireAuth(c)
	})
	v1Admin.GET("/genbodyforcluster", func(c *gin.Context) {
		genbody.GenBodyForCreateCluster(c)
	})

	v1Admin.POST("/kubeconf/:id", func(c *gin.Context) {
		genkubeconfig.GenKubeConfig(c)
	})

	v1Admin.GET("/auth", func(c *gin.Context) {
		middleware.Validate(c)
	})

	v1Admin.GET("/getusers", func(c *gin.Context) {
		user.FindAllUsers(c)
	})

	v1Admin.GET("/getuser/:id", func(c *gin.Context) {
		user.FindUserById(c)
	})

	v1Admin.PUT("/updateuser/:id", func(c *gin.Context) {
		user.UpdateUser(c)
	})

	v1Admin.DELETE("/deleteuser/:id", func(c *gin.Context) {
		user.DeleteUser(c)
	})
	v1Admin.GET("/mapsupercluster", func(c *gin.Context) {
		cluster.MapSuperClusters(c)
	})
}
