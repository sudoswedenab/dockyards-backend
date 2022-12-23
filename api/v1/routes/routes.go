package routes

import (
	"Backend/api/v1/handlers"
	jwt "Backend/api/v1/handlers/Jwt"
	"Backend/api/v1/handlers/user"
	"Backend/api/v1/middleware"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello Server",
		})
	})

	routes := r.Group("/")
	routes.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Slash API",
		})
	})

	apione := r.Group("/v1")

	apione.POST("/signup", func(c *gin.Context) {
		handlers.Signup(c)
	})

	apione.POST("/login", func(c *gin.Context) {
		handlers.Login(c)
	})

	apione.POST("/logout", func(c *gin.Context) {
		user.Logout(c)
	})

	apione.POST("/refresh", func(c *gin.Context) {
		err := jwt.RefreshTokenEndpoint(c)
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err))
		}
	})

	// Admin Routes
	admin := r.Group("/admin", func(c *gin.Context) {
		// Handles errors
		middleware.RequireAuth(c)
	})

	admin.GET("/auth", func(c *gin.Context) {
		handlers.Validate(c)
	})

	admin.GET("/getusers", func(c *gin.Context) {
		user.FindAllUsers(c)
	})

	admin.GET("/getuser/:id", func(c *gin.Context) {
		user.FindUserById(c)
	})

	admin.PUT("/updateuser/:id", func(c *gin.Context) {
		user.UpdateUser(c)
	})

	admin.DELETE("/deleteuser/:id", func(c *gin.Context) {
		user.DeleteUser(c)
	})
}
