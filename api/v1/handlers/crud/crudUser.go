package crud

import (
	"Backend/api/v1/models"
	"Backend/internal"

	"github.com/gin-gonic/gin"
)

// Find all users in DB
func FindAllUsers(c *gin.Context) {

	var users []models.User

	internal.DB.Find(&users)

	c.JSON(200, gin.H{
		"users": users,
	})
}

// Find user by Id
func FindUserById(c *gin.Context) {
	//Get Id off url
	id := c.Param("id")
	//get the User
	var userbyid models.User
	internal.DB.First(&userbyid, id)
	//Respond
	c.JSON(200, gin.H{
		"users": userbyid,
	})

}

// Update User
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

	c.Bind(&User)
	//find the post were updating
	var user models.User
	internal.DB.First(&user, id)
	//update it
	internal.DB.Model(&user).Updates(models.User{
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

// Delete User
func DeleteUser(c *gin.Context) {
	//Get the id off the url
	id := c.Param("id")
	//delete the post
	internal.DB.Delete(&models.User{}, id)
	//respond
	c.Status(200)

}

func Logout(c *gin.Context) {
	c.SetCookie("Authorization", "", -1, "", "", false, true)
}
