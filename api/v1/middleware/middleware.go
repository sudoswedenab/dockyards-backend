package middleware

import (
	"bitbucket.org/sudosweden/backend/internal/rancher"
	"gorm.io/gorm"
)

type Handler struct {
	DB             *gorm.DB
	RancherService rancher.RancherService
}
