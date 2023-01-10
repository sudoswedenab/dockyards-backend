package user

import (
	"Backend/api/v1/model"
	"Backend/internal"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// FindAllUsers godoc
//
//	@Summary		Lists all users
//	@Tags			FindAllUsers
//	@Accept       	application/json
//	@Produce		application/json
//	@Success		200	{array}	model.User
//	@Router			/admin/getusers [get]
func FindAllUsers(c *gin.Context) {
	var users []model.User

	internal.DB.Find(&users)

	c.JSON(200, gin.H{
		"user": users,
	})
}

// FindUserById godoc
//
//	@Summary		Find a user
//	@Tags			FindUserById
//	@Accept       	application/json
//	@Produce		application/json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	model.User
//	@Router			/admin/getuser/{id} [get]
func FindUserById(c *gin.Context) {
	//Get Id off url
	id := c.Param("id")
	//get the User
	var userById model.User
	internal.DB.First(&userById, id)
	//Respond
	c.JSON(200, gin.H{
		"user": userById,
	})
}

// UpdateUser godoc
//
//	@Summary		Update a user
//	@Tags			UpdateUser
//	@Accept       	application/json
//	@Produce		application/json
//	@Param			id	path		int	true	"User ID"
//	@Param			request	body	model.User	true "User model"
//	@Success		200	{object}	model.User
//	@Router			/admin/updateuser/{id} [put]
func UpdateUser(c *gin.Context) {
	//Get id of url
	id := c.Param("id")
	//Get the data off req body
	var User struct {
		Idn      int
		Name     string
		Email    string
		Password string
	}

	err := c.Bind(&User)
	if err != nil {
		fmt.Printf("Error: %s", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(User.Password), 10)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to hash password",
		})
		return
	}
	//find the post were updating
	var user model.User
	internal.DB.First(&user, id)
	//update it
	internal.DB.Model(&user).Updates(model.User{
		Idn:      User.Idn,
		Name:     User.Name,
		Email:    User.Email,
		Password: string(hash),
	})
	// Respond with it
	c.JSON(200, gin.H{
		"user": user,
	})
}

// DeleteUser godoc
//
//	@Summary		Delete a user
//	@Tags			DeleteUser
//	@Accept       	application/json
//	@Produce		text/plain
//	@Param			id	path	int	true	"User ID"
//	@Success		200
//	@Router			/admin/deleteuser/{id} [delete]
func DeleteUser(c *gin.Context) {
	//Get the id off the url
	id := c.Param("id")
	//delete the post
	internal.DB.Delete(&model.User{}, id)
	//respond
	c.Status(200)
}
