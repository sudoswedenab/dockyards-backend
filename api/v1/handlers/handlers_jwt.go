package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) PostRefresh(c *gin.Context) {
	ctx := context.Background()

	authorizationHeader := c.GetHeader("Authorization")
	if authorizationHeader == "" {
		h.logger.Debug("empty or missing authorization header during refresh")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	refreshToken := strings.TrimPrefix(authorizationHeader, "Bearer ")

	// Parse the token string and a function for looking for the key.
	token, err := jwt.ParseWithClaims(refreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return h.jwtRefreshTokenSecret, nil
	})
	if err != nil {
		h.logger.Error("error parsing token with claims", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		h.logger.Error("invalid token claims")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := claims.GetSubject()
	if err != nil {
		h.logger.Error("error getting subject from claims", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	matchingFields := client.MatchingFields{
		"metadata.uid": subject,
	}

	var userList v1alpha1.UserList
	err = h.controllerClient.List(ctx, &userList, matchingFields)
	if err != nil {
		h.logger.Error("", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if len(userList.Items) != 1 {
		h.logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	user := userList.Items[0]

	tokens, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("error generating tokens", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

func (h *handler) generateTokens(user v1alpha1.User) (*v1.Tokens, error) {
	claims := jwt.RegisteredClaims{
		Subject:   string(user.UID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedAccessToken, err := token.SignedString(h.jwtAccessTokenSecret)
	if err != nil {
		return nil, err
	}

	refreshTokenClaims := jwt.RegisteredClaims{
		Subject:   string(user.UID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 2)),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	signedRefreshToken, err := refreshToken.SignedString(h.jwtRefreshTokenSecret)
	if err != nil {
		return nil, err
	}

	tokens := v1.Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	return &tokens, nil
}
