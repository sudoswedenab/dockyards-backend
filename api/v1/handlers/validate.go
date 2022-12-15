package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Validate(c *gin.Context) {
	r := Response(c)

	// fmt.Println(r.Name)

	// c.JSON(http.StatusOK, gin.H{
	// 	"message": r.Name,
	// })

	println("AUTH hit")

	c.JSON(http.StatusOK, gin.H{
		"hey": "user logged in",
	})

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   gin.H{"user": r},
	})
}

// Get out info about user:
// user, _ := c.Get("user")
// fmt.Printf("%T\n", user.(models.User).CreatedAt)

// c.JSON(http.StatusOK, gin.H{

// 	"1.Name":      user.(models.User).Name,
// 	"2.Email":     user.(models.User).Email,
// 	"3.CreatedAt": user.(models.User).CreatedAt,
// 	"4.UpdatedAt": user.(models.User).UpdatedAt,
// })}
