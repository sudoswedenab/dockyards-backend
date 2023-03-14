package handlers

import (
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/rancher"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handler struct {
	db               *gorm.DB
	rancherService   rancher.RancherService
	accessTokenName  string
	refreshTokenName string
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rancherService rancher.RancherService) {
	h := handler{
		db:               db,
		rancherService:   rancherService,
		accessTokenName:  internal.AccessTokenName,
		refreshTokenName: internal.RefreshTokenName,
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)
	r.POST("/v1/logout", h.Logout)
}
