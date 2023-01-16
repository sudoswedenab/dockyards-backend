package handlers

import (
	"Backend/api/v1/handlers/rancher"
	"Backend/api/v1/model"
	"Backend/internal"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Signup godoc
//
//	@Summary		Signup to app
//	@Tags				Login
//	@Accept			application/json
//	@Produce		text/plain
//	@Param			request	body	model.Signup	true "Signup model"
//	@Success		201
//	@Router			/signup [post]
func Signup(c *gin.Context) {
	fmt.Println("Signup hit")
	var body model.Signup

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

	RancherID := rancher.RancherCreateUser(c, model.RancherUser{
		Enabled:            true,
		MustChangePassword: false,
		Name:               body.Name,
		Password:           body.Password,
		Username:           body.Email,
	})

	// fmt.Println(Kalle)
	//Create the user
	user := model.User{
		Name:      body.Name,
		Email:     body.Email,
		RancherID: RancherID,
		Password:  string(hash)}
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
