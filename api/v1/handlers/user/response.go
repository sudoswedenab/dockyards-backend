package user

import (
	"Backend/api/v1/model"
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
		Name:      user.(model.User).Name,
		Email:     user.(model.User).Email,
		CreatedAt: user.(model.User).CreatedAt,
		UpdatedAt: user.(model.User).UpdatedAt,
	}
}
