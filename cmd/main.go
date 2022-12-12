package main

import (
	"Backend/internal"

	"github.com/gin-gonic/gin"
)

func init() {
	internal.LoadEnvVariables()
}

func main() {

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World, Im Alive",
		})
	})

	r.Run()
}
