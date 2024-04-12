package middleware

import (
	"crypto/ecdsa"
	"log/slog"
)

type Handler struct {
	Logger          *slog.Logger
	AccessPublicKey *ecdsa.PublicKey
}
