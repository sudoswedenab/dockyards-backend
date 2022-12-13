package routes

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {

	///http://localhost:9000/api
	routes := r.Group("/")
	routes.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Im Alive",
		})
	})

	///http://localhost:9000/v1/login//
	apione := r.Group("/v1")
	apione.GET("/login", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Im Alive",
		})
	})
}
