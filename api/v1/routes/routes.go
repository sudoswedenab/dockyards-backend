package routes

import (
	"Backend/api/v1/handlers"
	"Backend/api/v1/handlers/crud"
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
		handlers.Signup(c)
	})

	apione.POST("/login", func(c *gin.Context) {
		handlers.Login(c)
	})

	apione.GET("/auth", func(c *gin.Context) {
		middleware.RequireAuth(c)
		handlers.Validate(c)
	})

	//http://localhost:9000/admin/getuser/1 exmpl
	admin := r.Group("/admin")

	admin.GET("/getusers", func(c *gin.Context) {
		middleware.RequireAuth(c)
		crud.FindAllUsers(c)
	})

	admin.GET("/getuser/:id", func(c *gin.Context) {
		middleware.RequireAuth(c)
		crud.FindUserById(c)
	})

	admin.PUT("/updateuser/:id", func(c *gin.Context) {
		middleware.RequireAuth(c)
		crud.UpdateUser(c)
	})

	admin.DELETE("/deleteuser/:id", func(c *gin.Context) {
		middleware.RequireAuth(c)
		crud.DeleteUser(c)
	})

}
