package handlers

import (
	"context"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) Login(c *gin.Context) {
	ctx := context.Background()

	var body v1.Login
	if c.BindJSON(&body) != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		"spec.email": body.Email,
	}

	var userList v1alpha1.UserList
	err := h.controllerClient.List(ctx, &userList, matchingFields)
	if err != nil {
		h.logger.Error("error getting user from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if len(userList.Items) != 1 {
		h.logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	user := userList.Items[0]

	if !user.Status.Verified {
		h.logger.Error("user has not been verified")

		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	//Compare sent in pass with saved user pass hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(body.Password))
	if err != nil {
		h.logger.Error("error comparing password", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	//Generate a jwt token
	accessToken := jwt.New(jwt.SigningMethodHS256)
	claims := accessToken.Claims.(jwt.MapClaims)
	claims["sub"] = user.UID
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// Sign and get the complete encoded token as a string using the secret
	at, err := accessToken.SignedString([]byte(h.jwtAccessTokenSecret))

	if err != nil {
		h.logger.Error("error signing access token", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)

	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.UID
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, err := refreshToken.SignedString([]byte(h.jwtRefreshTokenSecret))

	if err != nil {
		h.logger.Error("error signing refresh token", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tokens := v1.Tokens{
		AccessToken:  at,
		RefreshToken: rt,
	}

	c.JSON(http.StatusOK, tokens)
}
