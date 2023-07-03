package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"

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

	details, validName := names.IsValidName(body.Name)
	if !validName {
		h.logger.Error("invalid name in signup request", "name", body.Name, "details", details)

		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "singnup name is not valid",
			"name":    body.Name,
			"details": details,
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

	//Create the user
	user := model.User{
		Name:     body.Name,
		Email:    body.Email,
		Password: string(hash),
	}
	result := h.db.Create(&user)

	if result.Error != nil {
		h.logger.Error("error creating user in database", "err", err)

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
