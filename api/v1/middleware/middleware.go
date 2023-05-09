package middleware

import (
	"golang.org/x/exp/slog"
	"gorm.io/gorm"
)

type Handler struct {
	DB               *gorm.DB
	Logger           *slog.Logger
	Secret           string
	RefSecret        string
	AccessTokenName  string
	RefreshTokenName string
}
