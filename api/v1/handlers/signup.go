package handlers

import (
	"errors"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (h *handler) Signup(c *gin.Context) {
	var signup v1.Signup
	err := c.BindJSON(&signup)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	details, validName := names.IsValidName(signup.Name)
	if !validName {
		h.logger.Error("invalid name in signup request", "name", signup.Name, "details", details)

		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "signup name is not valid",
			"name":    signup.Name,
			"details": details,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(signup.Password), 10)
	if err != nil {
		h.logger.Error("error hashing password", "err", err)

		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "failed to hash password",
		})
		return
	}

	user := v1.User{
		ID:       uuid.New(),
		Name:     signup.Name,
		Email:    signup.Email,
		Password: string(hash),
	}

	err = h.db.Create(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "email or name is already in-use, reserved or forbidden",
				"name":  signup.Name,
				"email": signup.Email,
			})
			return
		}

		h.logger.Error("error creating user in database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	user.Password = ""

	c.JSON(http.StatusCreated, user)
}
