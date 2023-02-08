package routes

import (
	"Backend/api/v1/handlers"
	jwt "Backend/api/v1/handlers/Jwt"
	"Backend/api/v1/handlers/cluster"
	"Backend/api/v1/handlers/user"
	"Backend/api/v1/middleware"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Slash API",
		})
	})

	v1 := r.Group("/v1")

	v1.POST("/signup", func(c *gin.Context) {
		handlers.Signup(c)
	})

	v1.POST("/login", func(c *gin.Context) {
		handlers.Login(c)
	})

	v1.POST("/logout", func(c *gin.Context) {
		handlers.Logout(c)
	})

	v1.POST("/refresh", func(c *gin.Context) {
		err := jwt.RefreshTokenEndpoint(c)
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err))
		}
		c.String(http.StatusOK, "Success.")
	})

	v1.GET("/readcookie", func(c *gin.Context) {
		handlers.ReadCookie(c)
	})

	v1.GET("/mapcluster", func(c *gin.Context) {
		cluster.MapGetClusters(c)
	})

	v1.POST("/createcluster", func(c *gin.Context) {
		name, id, err := cluster.CreatedCluster(c)
		cluster.CreatedNodePool(c, name, id, err)
	})

	// Admin Routes
	v1Admin := v1.Group("/admin", func(c *gin.Context) {
		// Handles errors
		middleware.RequireAuth(c)
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
