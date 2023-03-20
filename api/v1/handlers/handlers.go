package handlers

import (
	"bitbucket.org/sudosweden/backend/internal"
	"bitbucket.org/sudosweden/backend/internal/rancher"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type handler struct {
	db               *gorm.DB
	rancherService   rancher.RancherService
	accessTokenName  string
	refreshTokenName string
	logger           *slog.Logger
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rancherService rancher.RancherService, logger *slog.Logger) {
	h := handler{
		db:               db,
		rancherService:   rancherService,
		accessTokenName:  internal.AccessTokenName,
		refreshTokenName: internal.RefreshTokenName,
		logger:           logger,
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)
	r.POST("/v1/logout", h.Logout)
}
