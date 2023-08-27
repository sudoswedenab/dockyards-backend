package user

import (
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"

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
		Name:      user.(v1.User).Name,
		Email:     user.(v1.User).Email,
		CreatedAt: *user.(v1.User).CreatedAt,
		UpdatedAt: *user.(v1.User).UpdatedAt,
	}
}
