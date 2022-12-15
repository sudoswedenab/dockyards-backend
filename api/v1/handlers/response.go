package handlers

import (
	"Backend/api/v1/models"
	"time"

	"github.com/gin-gonic/gin"
)

type userResponse struct {
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func Response(c *gin.Context) userResponse {
	user, _ := c.Get("user")
	return userResponse{
		Name:      user.(models.User).Name,
		Email:     user.(models.User).Email,
		CreatedAt: user.(models.User).CreatedAt,
		UpdatedAt: user.(models.User).UpdatedAt,
	}
}

// 		"Name":      user.(models.User).Name,
// 		"Email":     user.(models.User).Email,
// 		"CreatedAt": user.(models.User).CreatedAt,
// 		"UpdatedAt": user.(models.User).UpdatedAt,
