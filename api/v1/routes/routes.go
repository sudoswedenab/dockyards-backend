package routes

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/handlers"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, clusterService types.ClusterService) {
	middlewareHandler := middleware.Handler{
		DB: db,
	}

	r.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Slash API",
		})
	})

	v1 := r.Group("/v1")

	// Only used for checking server-side cookies
	v1.GET("/readcookie", func(c *gin.Context) {
		handlers.ReadCookie(c)
	})

	///
	// Admin Routes
	v1Admin := v1.Group("/admin", func(c *gin.Context) {
		// Handles errors
		middlewareHandler.RequireAuth(c)
	})

	v1Admin.GET("/auth", func(c *gin.Context) {
		handlers.Validate(c)
	})
}
