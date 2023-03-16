package handlers

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// Login godoc
//
//	@Summary		Login to app
//	@Tags			Login
//	@Accept			application/json
//	@Produce		text/plain
//	@Param			request	body	model.Login	true "Login model"
//	@Success		200
//	@Failure		400
//	@Router			/login [post]
func (h *handler) Login(c *gin.Context) {
	// Get email and pass off req body
	var body model.Login

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	//Look up requested User
	var user model.User

	h.db.First(&user, "email = ?", body.Email)

	if user.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email or password",
		})
		return
	}
	//Compare sent in pass with saved user pass hash
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bad hash or encryption",
		})
		return
	}
	//Checking agianst Racnher if user exist in rancher
	bearertoken, err := h.rancherService.RancherLogin(user)
	if err != nil {
		fmt.Printf("unexpected error doing user login in rancher: %s\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	//Generate a jwt token
	accessToken := jwt.New(jwt.SigningMethodHS256)
	claims := accessToken.Claims.(jwt.MapClaims)
	claims["aud"] = bearertoken
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// Sign and get the complete encoded token as a string using the secret
	at, err := accessToken.SignedString([]byte(internal.Secret))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create Token",
		})
		return
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)

	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.ID
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, rerr := refreshToken.SignedString([]byte(internal.RefSecret))

	if rerr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create Token",
		})
		return
	}

	if internal.FlagServerCookie {
		// Send back a Cookie
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(internal.AccessTokenName, at, 900, "", "", false, true)
		c.SetCookie(internal.RefreshTokenName, rt, 3600*1, "", "", false, true)
		c.JSON(http.StatusOK, gin.H{
			"Login": "Success",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Login":                   "Success",
			internal.AccessTokenName:  at,
			internal.RefreshTokenName: rt,
		})
	}
}
