package routes

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {

	routes := r.Group("/")
	routes.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Im Alive",
		})
	})
}
