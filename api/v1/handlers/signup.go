package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (h *handler) Signup(c *gin.Context) {
	var body model.Signup

	if c.BindJSON(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to hash password",
		})
		return
	}

	if err != nil {
		h.logger.Debug("unxepected error creating user in rancher", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}

	//Create the user
	user := model.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: string(hash),
	}
	result := h.db.Create(&user)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create User",
		})
		return
	}

	//respond
	c.JSON(http.StatusCreated, gin.H{
		"status": "You have now created your account",
	})
}
