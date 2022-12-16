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

}

// Delete User
func DeleteUser(c *gin.Context) {

}
