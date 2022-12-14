package handlers

import (
	"Backend/api/v1/models"
	"Backend/internal"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Signup(c *gin.Context) {
	println("Signup hit")

	//Get the email/pass req body

	var body struct {
		email    string
		password string
	}

	if c.Bind(&body) != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	//Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.password), 10)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to hash password",
		})
		return
	}
	//Create the user
	user := models.User{Email: body.email, Password: string(hash)}
	result := internal.DB.Create(&user)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create User",
		})
		return
	}

	//respond
	c.JSON(http.StatusOK, gin.H{})
}
