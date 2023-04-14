package handlers

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func (h *handler) PostRefresh(c *gin.Context) {
	// Get the cookie
	refreshToken, err := c.Cookie(internal.RefreshTokenName)

	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}

	// Parse the token string and a function for looking for the key.
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your incl secret key
		return []byte(internal.RefSecret), nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get the user record from database or
		// why float64 and not int64?
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		//Find the user with token sub
		var user model.User

		First := h.db.First(&user, claims["sub"])

		// replace with jwt response
		if First.Error == nil {
			newTokenPair, err := h.generateTokenPair(user)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}
			c.JSON(http.StatusOK, gin.H{
				"Login":                   "Success",
				internal.AccessTokenName:  newTokenPair[internal.AccessTokenName],
				internal.RefreshTokenName: newTokenPair[internal.RefreshTokenName],
			})
		}
	}
}

func (h *handler) generateTokenPair(user model.User) (map[string]string, error) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(internal.Secret))
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.ID
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, err := refreshToken.SignedString([]byte(internal.RefSecret))
	if err != nil {
		return nil, err
	}

	return map[string]string{
		internal.AccessTokenName:  t,
		internal.RefreshTokenName: rt,
	}, nil
}
