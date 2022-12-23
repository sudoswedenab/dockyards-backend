package handlers

import (
	"Backend/api/v1/model"
	"Backend/internal"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Signup godoc
//
//	@Summary		Signup
//	@Description	signup to api
//	@Tags			Signup
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Account ID"
//	@Success		200	{object}	model.User
//	@Router			/signup [post]
func Signup(c *gin.Context) {
	println("Sign hit")

	//Get the email/pass req body

	var body struct {
		Email    string
		Password string
		Name     string
	}

	if c.Bind(&body) != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	//Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to hash password",
		})
		return
	}
	//Create the user
	user := model.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: string(hash)}
	result := internal.DB.Create(&user)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create User",
		})
		return
	}

	//respond

	c.JSON(http.StatusCreated, gin.H{
		"status": "You have now created ure account",
	})
}
