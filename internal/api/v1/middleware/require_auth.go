package middleware

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type RequireAuth struct {
	publicKey *ecdsa.PublicKey
}

func (a *RequireAuth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := LoggerFrom(r.Context())

		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			logger.Debug("empty or missing authorization header", "method", r.Method, "path", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		bearerToken := strings.TrimPrefix(authorizationHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(bearerToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodECDSA)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return a.publicKey, nil
		})
		if err != nil {
			logger.Error("error parsing bearer token", "err", err)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		claims, ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok && !token.Valid {
			logger.Debug("invalid claims")
			w.WriteHeader(http.StatusUnauthorized)
		}

		subject, err := claims.GetSubject()
		if err != nil {
			logger.Debug("error getting subject from claim", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		ctx := context.WithValue(r.Context(), sub, subject)

		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
	})
}

func ContextWithSubject(parent context.Context, subject string) context.Context {
	return context.WithValue(parent, sub, subject)
}

func SubjectFrom(ctx context.Context) (string, error) {
	v := ctx.Value(sub)
	if v == nil {
		return "", errors.New("error fecthing subject from context")
	}

	sub, ok := v.(string)
	if !ok {
		return "", errors.New("error during type conversion")
	}

	return sub, nil
}

func NewRequireAuth(publicKey *ecdsa.PublicKey) *RequireAuth {
	a := RequireAuth{
		publicKey: publicKey,
	}

	return &a
}
