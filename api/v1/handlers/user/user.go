package user

import (
	"Backend/api/v1/model"
	"Backend/internal"
	"fmt"

	"github.com/gin-gonic/gin"
)

// FindAllUsers godoc
//
//	@Summary		FindAllUsers find all users
//	@Description	gets all users
//	@Tags			FindAllUsers
//	@Accept       	json
//	@Produce		json
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
//	@Summary		FindUserById finds user
//	@Description	gets specified user
//	@Tags			FindUserById
//	@Accept       	json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	model.User
//	@Router			/getuser/{id} [get]
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
//	@Summary		UpdateUser update user
//	@Description	updates specified user
//	@Tags			UpdateUser
//	@Accept       	json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	model.User
//	@Router			/updateuser/{id} [put]
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
	//find the post were updating
	var user model.User
	internal.DB.First(&user, id)
	//update it
	internal.DB.Model(&user).Updates(model.User{
		Idn:      User.Idn,
		Name:     User.Name,
		Email:    User.Email,
		Password: User.Password,
	})
	// Respond with it
	c.JSON(200, gin.H{
		"user": user,
	})
}

// DeleteUser godoc
//
//	@Summary		DeleteUser delete user
//	@Description	deletes specified user
//	@Tags			DeleteUser
//	@Accept       	json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{code}		200
//	@Router			/deleteuser/{id} [delete]
func DeleteUser(c *gin.Context) {
	//Get the id off the url
	id := c.Param("id")
	//delete the post
	internal.DB.Delete(&model.User{}, id)
	//respond
	c.Status(200)
}

func Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "", "", false, true)
	c.SetCookie("refresh_token", "", -1, "", "", false, true)
}
