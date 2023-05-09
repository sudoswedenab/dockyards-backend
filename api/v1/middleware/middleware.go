package middleware

import (
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type Handler struct {
	DB                 *gorm.DB
	Logger             *slog.Logger
	AccessTokenSecret  string
	RefreshTokenSecret string
	AccessTokenName    string
	RefreshTokenName   string
}
