package routes

import (
	"Backend/api/v1/handlers"
	"Backend/api/v1/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello Server",
		})
	})

	///http://localhost:9000/api
	routes := r.Group("/")
	routes.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Slash APi",
		})
	})

	///http://localhost:9000/v1/login/
	apione := r.Group("/v1")

	apione.POST("/signup", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Sign-Up information",
		})
		handlers.Signup(c)
	})

	apione.POST("/login", func(c *gin.Context) {
		handlers.Login(c)
	})

	apione.GET("/auth", func(c *gin.Context) {
		middleware.RequireAuth(c)
		handlers.Validate(c)
	})
}
