package handlers

import (
	"bitbucket.org/sudosweden/backend/internal/rancher"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handler struct {
	db             *gorm.DB
	rancherService rancher.RancherService
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rancherService rancher.RancherService) {
	h := handler{
		db:             db,
		rancherService: rancherService,
	}

	r.POST("/v1/signup", h.Signup)
	r.POST("/v1/login", h.Login)
}
