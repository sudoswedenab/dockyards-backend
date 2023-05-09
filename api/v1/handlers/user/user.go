package user

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type handler struct {
	db *gorm.DB
}

func (h *handler) FindAllUsers(c *gin.Context) {
	var users []model.User

	h.db.Find(&users)

	c.JSON(200, gin.H{
		"user": users,
	})
}

func (h *handler) FindUserById(c *gin.Context) {
	//Get Id off url
	id := c.Param("id")
	//get the User
	var userById model.User
	h.db.First(&userById, "id = ?", id)
	//Respond
	c.JSON(200, gin.H{
		"user": userById,
	})
}

func (h *handler) UpdateUser(c *gin.Context) {
	//Get id of url
	id := c.Param("id")
	//Get the data off req body
	var User struct {
		Idn      int
		Name     string
		Email    string
		Password string
	}

	err := c.BindJSON(&User)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
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
	h.db.First(&user, id)
	//update it
	h.db.Model(&user).Updates(model.User{
		Name:     User.Name,
		Email:    User.Email,
		Password: string(hash),
	})
	// Respond with it
	c.JSON(200, gin.H{
		"user": user,
	})
}

func (h *handler) DeleteUser(c *gin.Context) {
	//Get the id off the url
	id := c.Param("id")
	//delete the post
	h.db.Delete(&model.User{}, "id = ?", id)
	//respond
	c.Status(200)
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	h := handler{
		db: db,
	}

	middlewareHandler := middleware.Handler{
		DB: db,
	}

	g := r.Group("/v1/admin")
	g.Use(middlewareHandler.RequireAuth)

	g.GET("/getusers", h.FindAllUsers)
	g.GET("/getuser/:id", h.FindUserById)
	g.PUT("/updateuser/:id", h.UpdateUser)
	g.DELETE("/deleteuser/:id", h.DeleteUser)
}
