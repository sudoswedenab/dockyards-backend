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

	// Remove?
	a := r.Group("/")
	a.GET("/api", func(c *gin.Context) {
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
		c.String(http.StatusOK, fmt.Sprintf("Success."))
	})

	// Admin Routes
	v1Admin := v1.Group("/admin", func(c *gin.Context) {
		// Handles errors
		middleware.RequireAuth(c)
	})

	v1Admin.GET("/auth", func(c *gin.Context) {
		handlers.Validate(c)
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
}
